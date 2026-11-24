package poller

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"contrail/internal/config"
	"contrail/internal/database"
	"contrail/internal/models"
)

// Poller orchestrates multi-provider ingestion, dead reckoning, enrichment, and persistence.
type Poller struct {
	cfg              *config.Config
	db               *database.DB
	openSky          *OpenSkyScheduler
	adsbMesh         *ADSBMeshScheduler
	enricher         *RouteEnricher
	aircraftEnricher *AircraftEnricher
	diversionEngine  *DiversionEngine
	airportDB        *AirportDB
	sourcePriority   map[string]int
	liveCache        map[string]*models.FlightData
	lastPollTime     time.Time
	lastFlightCount  int
	mu               sync.RWMutex
}

// NewPoller initializes the Poller and independent schedulers.
func NewPoller(cfg *config.Config, db *database.DB) *Poller {
	airportDB := GetAirportDB()
	enricher := NewRouteEnricher(db, cfg.ADSBDbBaseURL, cfg.ADSBDbRateLimitPerMin, cfg.CallsignCacheTTLHours)
	aircraftEnricher := NewAircraftEnricher(db, enricher.limiter)
	diversionEngine := NewDiversionEngine(airportDB)
	if db != nil {
		diversionEngine.SetDB(db)
	}

	// Build source priority mapping (lower rank number = higher priority)
	priMap := make(map[string]int)
	for idx, s := range cfg.SourcePriority {
		priMap[strings.ToLower(strings.TrimSpace(s))] = idx
	}
	if len(priMap) == 0 {
		priMap["opensky"] = 0
		priMap["adsb_mesh"] = 1
	}

	p := &Poller{
		cfg:              cfg,
		db:               db,
		enricher:         enricher,
		aircraftEnricher: aircraftEnricher,
		diversionEngine:  diversionEngine,
		airportDB:        airportDB,
		sourcePriority:   priMap,
		liveCache:        make(map[string]*models.FlightData),
	}

	// Instantiate independent schedulers
	p.openSky = NewOpenSkyScheduler(cfg, db, p.IngestBatch)
	p.adsbMesh = NewADSBMeshScheduler(cfg, db, p.IngestBatch)

	// Wire unverified diversion feedback into adsb.lol targeted calibration queue
	p.diversionEngine.SetOnUnverified(func(hex string) {
		p.adsbMesh.EnqueueCalibration(hex, 2)
	})

	// Wire accuracy evaluation hook
	p.adsbMesh.SetAccuracyHook(func(hex string, obsLat, obsLon, obsAltFt, obsTrack float64, lastContact int64) {
		p.mu.RLock()
		existing, ok := p.liveCache[hex]
		p.mu.RUnlock()
		if ok && existing != nil && existing.PositionSource == "estimated" {
			prevLat := existing.Telemetry.Position.Latitude
			prevLon := existing.Telemetry.Position.Longitude
			errKm := HaversineDistance(prevLat, prevLon, obsLat, obsLon)
			ageSec := int(lastContact - existing.Metadata.LastObservedAt)
			if ageSec < 0 {
				ageSec = 0
			}
			_ = p.db.RecordExtrapolationAccuracy(hex, time.Now().Unix(), prevLat, prevLon, obsLat, obsLon, errKm, ageSec)
		}
	})

	if db != nil {
		db.SetMaxAgeResolver(p.GetDynamicMaxAgeSec)
	}

	return p
}

// Start launches concurrent schedulers, enricher workers, and the background retention cleaner.
func (p *Poller) Start(ctx context.Context) {
	log.Printf("[Poller] Starting multi-upstream aviation telemetry engine (Mode: %s)...", p.cfg.UpstreamProvider)
	p.enricher.StartWorker(ctx, 5)
	p.aircraftEnricher.StartWorker(ctx, 3)

	prov := strings.ToLower(strings.TrimSpace(p.cfg.UpstreamProvider))

	// Launch OpenSky independent scheduler
	if prov == "auto" || prov == "opensky" {
		p.openSky.Start(ctx)
	}

	// Launch ADS-B Mesh independent scheduler
	if prov == "auto" || prov == "mesh" || prov == "adsb_fi" {
		p.adsbMesh.Start(ctx)
	}

	// Background purge worker: removes flights not observed within AircraftPurgeTimeoutSec (default 10 minutes)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		purgeTimeout := time.Duration(p.cfg.AircraftPurgeTimeoutSec) * time.Second
		if purgeTimeout < 3*time.Minute {
			purgeTimeout = 10 * time.Minute
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				removed, err := p.db.RemoveStaleLiveFlights(purgeTimeout)
				if err == nil && removed > 0 {
					log.Printf("[Poller] Purged %d stale aircraft (no true contact for > %v)", removed, purgeTimeout)
				}
				p.diversionEngine.CleanupStale(30 * time.Minute)

				// Clean up in-memory cache
				cutoff := time.Now().Add(-purgeTimeout).Unix()
				p.mu.Lock()
				for hex, f := range p.liveCache {
					if f.Metadata.LastObservedAt < cutoff && f.Telemetry.LastContactEpoch < cutoff {
						delete(p.liveCache, hex)
					}
				}
				p.mu.Unlock()
			}
		}
	}()
}

// IngestBatch merges incoming real observed snapshots into the live pool.
// Partial or failed batches NEVER delete other aircraft.
func (p *Poller) IngestBatch(snaps []*models.RawAircraftSnapshot, source string) {
	if len(snaps) == 0 {
		return
	}

	now := time.Now().UTC()
	toPersist := make([]models.FlightData, 0, len(snaps))

	p.mu.Lock()
	for _, raw := range snaps {
		if raw.LastContact > 10000000000 {
			raw.LastContact /= 1000
		}
		hex := raw.Hex
		existing, exists := p.liveCache[hex]

		// Merging rule:
		// 1. Take the snapshot with newer last_contact.
		// 2. If identical last_contact, tiebreak by source priority.
		shouldUpdate := false
		if !exists {
			shouldUpdate = true
		} else if raw.LastContact > existing.Metadata.LastObservedAt {
			shouldUpdate = true
		} else if raw.LastContact == existing.Metadata.LastObservedAt {
			priNew := p.getSourcePriority(source)
			priExisting := p.getSourcePriority(existing.Metadata.UpstreamSource)
			if priNew <= priExisting {
				shouldUpdate = true
			}
		}

		if shouldUpdate {
			// Record extrapolation accuracy error if replacing an estimated position
			if exists && existing != nil && existing.PositionSource == "estimated" && raw.Lat != 0 && raw.Lon != 0 {
				prevLat := existing.Telemetry.Position.Latitude
				prevLon := existing.Telemetry.Position.Longitude
				errKm := HaversineDistance(prevLat, prevLon, raw.Lat, raw.Lon)
				ageSec := int(raw.LastContact - existing.Metadata.LastObservedAt)
				if ageSec < 0 {
					ageSec = 0
				}
				_ = p.db.RecordExtrapolationAccuracy(hex, now.Unix(), prevLat, prevLon, raw.Lat, raw.Lon, errKm, ageSec)
			}

			fd := p.processAircraft(raw, source, now)
			p.liveCache[hex] = &fd
			toPersist = append(toPersist, fd)
		}
	}
	p.lastPollTime = now
	p.lastFlightCount = len(p.liveCache)
	p.mu.Unlock()

	// Persist batch into DB (only real observed points are written to flight_snapshots)
	if len(toPersist) > 0 {
		if err := p.db.SaveLiveFlights(toPersist); err != nil {
			log.Printf("[Poller] Failed to persist %d live flights from '%s': %v", len(toPersist), source, err)
		}
	}
}

func (p *Poller) getSourcePriority(source string) int {
	s := strings.ToLower(strings.TrimSpace(source))
	if pri, ok := p.sourcePriority[s]; ok {
		return pri
	}
	return 99
}

func (p *Poller) processAircraft(raw *models.RawAircraftSnapshot, source string, now time.Time) models.FlightData {
	callsign := raw.Flight

	// 1. Enrich aircraft registration, type, and country
	reg, typeCode, country := p.aircraftEnricher.EnrichAircraft(raw.Hex, raw.Registration, raw.Type)

	// 2. Enrich route and airline from adsbdb cache / queue / embedded airlines DB
	route, airline, iataPtr := p.enricher.GetRoute(callsign)

	// 3. Sensor anomaly check:
	// If commercial flight reports altitude > 50,000 ft (service ceiling of airliners is ~41,000-43,000ft,
	// e.g. DLH797 sensor bit-flip reporting 53,939ft), sanitize to geometric altitude or 41,000ft ceiling.
	altBaro := raw.AltBaroFt
	if altBaro > 50000 {
		if raw.AltGeomFt > 1000 && raw.AltGeomFt <= 45000 {
			altBaro = raw.AltGeomFt
		} else {
			altBaro = 41000
		}
	}
	altGeom := raw.AltGeomFt
	if altGeom <= 0 {
		altGeom = altBaro
	}

	// 4. Identify and parse squawk / emergency status (real observation)
	emergInfo := p.parseEmergency(raw.Squawk, raw.Emergency)

	// 5. Compute diversion prediction on REAL observed data point
	diversionAnalysis := p.diversionEngine.Analyze(raw, route)

	// Build composite flight data
	var routeCopy models.RouteInfo
	if route != nil {
		routeCopy = *route
	}

	fd := models.FlightData{
		Aircraft: models.AircraftInfo{
			Hex:          raw.Hex,
			Registration: reg,
			Type:         typeCode,
			Category:     raw.Category,
			Country:      country,
		},
		Flight: models.FlightIdentifier{
			CallsignICAO: callsign,
			CallsignIATA: iataPtr,
			Airline:      airline,
		},
		Route: routeCopy,
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  math.Round(raw.Lat*100000) / 100000,
				Longitude: math.Round(raw.Lon*100000) / 100000,
			},
			Altitude: models.AltitudeInfo{
				BarometricFt: math.Round(altBaro),
				GeometricFt:  math.Round(altGeom),
				BarometricM:  math.Round(altBaro * 0.3048),
			},
			Speed: models.SpeedInfo{
				GroundSpeedKts: math.Round(raw.GroundSpeed*10) / 10,
				GroundSpeedKmh: math.Round(raw.GroundSpeed*1.852*10) / 10,
			},
			Attitude: models.AttitudeInfo{
				TrackDeg:        math.Round(raw.Track*10) / 10,
				VerticalRateFpm: math.Round(raw.VerticalRate),
				VerticalRateMps: math.Round(raw.VerticalRate*0.00508*100) / 100,
			},
			OnGround:         raw.OnGround,
			LastContactEpoch: raw.LastContact,
			PositionSource:   "observed",
			EstimatedAgeS:    0,
			LastObservedAt:   raw.LastContact,
		},
		Emergency:         emergInfo,
		DiversionAnalysis: diversionAnalysis,
		Metadata: models.MetadataInfo{
			UpstreamSource: source,
			LastUpdated:    now,
			PositionSource: "observed",
			EstimatedAgeS:  0,
			LastObservedAt: raw.LastContact,
		},
		PositionSource: "observed",
		EstimatedAgeS:  0,
		LastObservedAt: raw.LastContact,
	}

	return fd
}

func (p *Poller) parseEmergency(squawk, emergencyField string) models.EmergencyInfo {
	trimmedSq := strings.TrimSpace(squawk)
	trimmedEm := strings.ToLower(strings.TrimSpace(emergencyField))

	switch trimmedSq {
	case "7700":
		return models.EmergencyInfo{
			Squawk:      "7700",
			IsEmergency: true,
			Category:    models.EmergencyGeneral,
			Description: "Squawk 7700 declared - Mayday / General In-flight Emergency",
		}
	case "7600":
		return models.EmergencyInfo{
			Squawk:      "7600",
			IsEmergency: true,
			Category:    models.EmergencyRadioFailure,
			Description: "Squawk 7600 declared - Radio Communication Failure (NORDO)",
		}
	case "7500":
		return models.EmergencyInfo{
			Squawk:      "7500",
			IsEmergency: true,
			Category:    models.EmergencyHijack,
			Description: "Squawk 7500 declared - Unlawful Interference / Hijacking in Progress",
		}
	}

	if trimmedEm != "" && trimmedEm != "none" && trimmedEm != "0" {
		return models.EmergencyInfo{
			Squawk:      trimmedSq,
			IsEmergency: true,
			Category:    models.EmergencyUpstreamGeneral,
			Description: fmt.Sprintf("Upstream declared emergency state: %s", emergencyField),
		}
	}

	return models.EmergencyInfo{
		Squawk:      trimmedSq,
		IsEmergency: false,
		Category:    models.EmergencyNone,
		Description: "Normal operations",
	}
}

// GetDynamicMaxAgeSec calculates the dynamic dead reckoning max age:
// min(configured_max, upstream_current_interval * 1.2), default configured_max = 720s (12 minutes).
func (p *Poller) GetDynamicMaxAgeSec(source string) float64 {
	cfgMax := float64(p.cfg.DeadReckoningMaxAgeSec)
	if cfgMax <= 0 {
		cfgMax = 720.0
	}

	s := strings.ToLower(strings.TrimSpace(source))
	var upstreamInterval time.Duration
	if s == "opensky" && p.openSky != nil {
		upstreamInterval = p.openSky.GetCurrentInterval()
	} else if (s == "adsb_mesh" || s == "mesh") && p.adsbMesh != nil {
		upstreamInterval = p.adsbMesh.GetCurrentInterval()
	}

	if upstreamInterval > 0 {
		dyn := upstreamInterval.Seconds() * 1.2
		return math.Min(cfgMax, dyn)
	}

	return cfgMax
}

// GetStatus returns operational metrics and upstream health observability.
func (p *Poller) GetStatus() map[string]interface{} {
	p.mu.RLock()
	lastTime := p.lastPollTime
	flightCount := p.lastFlightCount
	p.mu.RUnlock()

	upstreams := map[string]interface{}{
		"opensky":   p.openSky.GetStatus(),
		"adsb_mesh": p.adsbMesh.GetStatus(),
	}

	return map[string]interface{}{
		"poller_active":     true,
		"last_poll_time":    lastTime.UTC().Truncate(time.Second),
		"last_flight_count": flightCount,
		"upstreams":         upstreams,
	}
}

// UpstreamPauseInfo holds pause status, reason, and estimated recovery time for the UI banner.
type UpstreamPauseInfo struct {
	IsPaused      bool      `json:"is_paused"`
	Reason        string    `json:"reason"`
	ResumeAt      time.Time `json:"resume_at,omitempty"`
	ResumeTimeStr string    `json:"resume_time_str,omitempty"`
}

// GetUpstreamPauseInfo computes whether all upstreams are currently in cooldown and formats the earlier recovery time.
func (p *Poller) GetUpstreamPauseInfo() UpstreamPauseInfo {
	openSkyStatus := p.openSky.GetStatus()
	adsbStatus := p.adsbMesh.GetStatus()

	now := time.Now()
	openSkyCooldown := false
	adsbCooldown := false
	var openSkyResume time.Time
	var adsbResume time.Time

	if cu, ok := openSkyStatus["cooldown_until"].(string); ok && cu != "" {
		if t, err := time.Parse(time.RFC3339, cu); err == nil && t.After(now) {
			openSkyCooldown = true
			openSkyResume = t
		}
	}
	if cu, ok := adsbStatus["cooldown_until"].(string); ok && cu != "" {
		if t, err := time.Parse(time.RFC3339, cu); err == nil && t.After(now) {
			adsbCooldown = true
			adsbResume = t
		}
	}

	if !openSkyCooldown && !adsbCooldown {
		return UpstreamPauseInfo{IsPaused: false}
	}

	var earliestResume time.Time
	var reasons []string
	if openSkyCooldown {
		reasons = append(reasons, "OpenSky 冷却中")
		earliestResume = openSkyResume
	}
	if adsbCooldown {
		reasons = append(reasons, "adsb.lol 冷却中")
		if earliestResume.IsZero() || adsbResume.Before(earliestResume) {
			earliestResume = adsbResume
		}
	}

	cstLoc := time.FixedZone("CST", 8*3600)
	timeStr := earliestResume.In(cstLoc).Format("15:04")

	return UpstreamPauseInfo{
		IsPaused:      true,
		Reason:        strings.Join(reasons, " / "),
		ResumeAt:      earliestResume,
		ResumeTimeStr: timeStr,
	}
}

// EnqueueCalibration adds an aircraft hex to the adsb.lol targeted calibration queue.
func (p *Poller) EnqueueCalibration(hex string, priority int) {
	if p.adsbMesh != nil {
		p.adsbMesh.EnqueueCalibration(hex, priority)
	}
}
