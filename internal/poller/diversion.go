package poller

import (
	"fmt"
	"math"
	"sync"
	"time"

	"contrail/internal/database"
	"contrail/internal/models"
)

// HistoryPoint captures a historical trajectory coordinate for holding pattern and deviation tracking.
type HistoryPoint struct {
	Lat   float64
	Lon   float64
	Track float64
	Time  time.Time
}

// AircraftTrackHistory tracks recent telemetry points for evaluating duration conditions.
type AircraftTrackHistory struct {
	LastDeviationTime time.Time
	DeviationCount    int
	LastAltitude      float64
	LastTimestamp     time.Time
	LastContactEpoch  int64
	Points            []HistoryPoint
}

// DiversionEngine computes real-time diversion predictions.
type DiversionEngine struct {
	mu           sync.RWMutex
	history      map[string]*AircraftTrackHistory
	airportDB    *AirportDB
	db           *database.DB
	onUnverified func(hex string)
}

// NewDiversionEngine creates a new diversion prediction engine.
func NewDiversionEngine(airportDB *AirportDB) *DiversionEngine {
	return &DiversionEngine{
		history:   make(map[string]*AircraftTrackHistory),
		airportDB: airportDB,
	}
}

// SetDB injects the database instance for corridor queries.
func (de *DiversionEngine) SetDB(db *database.DB) {
	de.mu.Lock()
	defer de.mu.Unlock()
	de.db = db
}

// SetOnUnverified sets the callback invoked when a flight is flagged as unverified diversion.
func (de *DiversionEngine) SetOnUnverified(fn func(hex string)) {
	de.mu.Lock()
	defer de.mu.Unlock()
	de.onUnverified = fn
}

// Analyze evaluates the diversion criteria for a given aircraft.
func (de *DiversionEngine) Analyze(raw *models.RawAircraftSnapshot, route *models.RouteInfo) models.DiversionAnalysis {
	analysis := models.DiversionAnalysis{
		PredictedDiversion: false,
		ProbabilityLevel:   "NONE",
		StatusTier:         "none",
		TriggeredCriteria:  make([]models.DiversionCriterion, 0),
		TriggerReasons:     make([]string, 0),
	}

	// 1. If aircraft is on ground, or raw contact is an estimated/stale point, return base analysis
	if raw.OnGround || raw.Source == "estimated" || raw.Source == "stale" {
		analysis.ProbabilityLevel = "NONE"
		analysis.StatusTier = "none"
		return analysis
	}

	// Record track history point for holding pattern and continuity
	now := time.Now()
	de.mu.Lock()
	hist, exists := de.history[raw.Hex]
	if !exists {
		hist = &AircraftTrackHistory{
			LastAltitude:  raw.AltBaroFt,
			LastTimestamp: now,
		}
		de.history[raw.Hex] = hist
	}
	hist.Points = append(hist.Points, HistoryPoint{
		Lat:   raw.Lat,
		Lon:   raw.Lon,
		Track: raw.Track,
		Time:  now,
	})
	if len(hist.Points) > 12 {
		hist.Points = hist.Points[len(hist.Points)-12:]
	}
	isHolding := detectHoldingPattern(hist.Points)
	de.mu.Unlock()

	// 2. Holding pattern detection: suppress diversion alerts for circular/racetrack maneuvers
	if isHolding {
		analysis.IsHolding = true
		analysis.PredictedDiversion = false
		analysis.ProbabilityLevel = "NONE"
		analysis.StatusTier = "none"
		analysis.TriggeredCriteria = append(analysis.TriggeredCriteria, models.DiversionCriterion{
			Code:      "HOLDING_PATTERN",
			Message:   "Holding pattern detected: 360-degree turn in localized area (< 20 km)",
			Triggered: true,
		})
		analysis.TriggerReasons = append(analysis.TriggerReasons, "Holding pattern: 360-degree turn in localized area (< 20 km)")
		return analysis
	}

	// 3. Transponder emergency squawk evaluation (Condition 4)
	isEmergencySquawk := raw.Squawk == "7700" || raw.Squawk == "7600" || raw.Squawk == "7500"
	if isEmergencySquawk {
		var desc string
		switch raw.Squawk {
		case "7700":
			desc = "Transponder set to emergency code 7700 (General Emergency)"
		case "7600":
			desc = "Transponder set to emergency code 7600 (Radio Failure)"
		case "7500":
			desc = "Transponder set to emergency code 7500 (Hijack / Unlawful Interference)"
		}
		analysis.TriggeredCriteria = append(analysis.TriggeredCriteria, models.DiversionCriterion{
			Code:      "EMERGENCY_SQUAWK",
			Message:   desc,
			Triggered: true,
		})
	}

	// 4. Without destination coordinates, evaluate emergency squawk alone
	if route == nil || route.Destination == nil || (route.Destination.Location.Latitude == 0 && route.Destination.Location.Longitude == 0) {
		if isEmergencySquawk {
			analysis.ProbabilityLevel = "MODERATE"
		}
		return analysis
	}

	// 5. Same origin and destination (circular/survey/test flight) does not participate in diversion analysis
	if route.Origin != nil {
		origICAO := route.Origin.ICAO
		destICAO := route.Destination.ICAO
		origIATA := route.Origin.IATA
		destIATA := route.Destination.IATA
		if (origICAO != "" && origICAO == destICAO) || (origIATA != "" && origIATA == destIATA) {
			if isEmergencySquawk {
				analysis.ProbabilityLevel = "MODERATE"
			}
			return analysis
		}

		// Route plausibility verification: check if cached adsbdb route is trustworthy for this active flight
		if !isEmergencySquawk {
			isPlausible, isReversed, reason := ValidateRoutePlausibility(raw, route)
			if isReversed {
				// Reverse flight leg detected: swap origin and destination
				route.Origin, route.Destination = route.Destination, route.Origin
			} else if !isPlausible {
				// Route contradicts aircraft's physical flight vector (callsign reuse or outdated schedule)
				analysis.ProbabilityLevel = "NONE"
				analysis.StatusTier = "none"
				analysis.PredictedDiversion = false
				analysis.TriggerReasons = append(analysis.TriggerReasons, "Route untrusted: "+reason)
				return analysis
			}
		}
	}

	destLat := route.Destination.Location.Latitude
	destLon := route.Destination.Location.Longitude

	distToDest := HaversineDistance(raw.Lat, raw.Lon, destLat, destLon)
	bearingToDest := InitialBearing(raw.Lat, raw.Lon, destLat, destLon)
	headingDev := AngularDifference(raw.Track, bearingToDest)

	analysis.DistanceToDestKm = math.Round(distToDest*10) / 10
	analysis.BearingToDestDeg = math.Round(bearingToDest*10) / 10
	analysis.HeadingDeviationDeg = math.Round(headingDev*10) / 10

	// Corridor Plausibility & Route Baseline Check
	if route.Origin != nil && route.Origin.Location.Latitude != 0 && route.Origin.Location.Longitude != 0 {
		origLat := route.Origin.Location.Latitude
		origLon := route.Origin.Location.Longitude
		routeDist := HaversineDistance(origLat, origLon, destLat, destLon)
		if routeDist >= 120.0 {
			distToOrig := HaversineDistance(raw.Lat, raw.Lon, origLat, origLon)
			if distToDest > routeDist*1.15 || distToOrig > routeDist*1.15 {
				// Flight is completely outside the geographic corridor of this route
				return analysis
			}

			// Check learned corridor or Great Circle path
			isCorridorDeviated, corridorMsg := de.checkCorridorDeviation(
				route.Origin.ICAO, route.Destination.ICAO,
				raw.Lat, raw.Lon, origLat, origLon, destLat, destLon,
			)
			if isCorridorDeviated {
				analysis.TriggeredCriteria = append(analysis.TriggeredCriteria, models.DiversionCriterion{
					Code:      "CORRIDOR_DEVIATION",
					Message:   corridorMsg,
					Triggered: true,
				})
			}
		}
	}

	// 6. Normal Destination Approach & Landing Check:
	// If within 40 km of scheduled destination, or below 3,000 ft and within 60 km of destination,
	// this is a normal arrival/landing at the destination, NOT a diversion!
	if distToDest <= 40.0 || (raw.AltBaroFt < 3000 && distToDest <= 60.0) {
		if isEmergencySquawk {
			analysis.ProbabilityLevel = "MODERATE"
		} else {
			analysis.ProbabilityLevel = "NONE"
		}
		analysis.PredictedDiversion = false
		analysis.StatusTier = "none"
		return analysis
	}

	// 7. Condition 1: Descending far from destination
	// Altitude < 12,000 ft, persistent negative vertical rate (< -500 ft/min), and distance to destination > 120 km
	isDescendingFar := raw.AltBaroFt < 12000 && raw.VerticalRate < -500 && distToDest > 120.0
	if isDescendingFar {
		analysis.TriggeredCriteria = append(analysis.TriggeredCriteria, models.DiversionCriterion{
			Code: "DESCENT_FAR_FROM_DEST",
			Message: fmt.Sprintf("Altitude %.0fft (< 12,000ft) and descent rate %.0ffpm while %.1fkm from destination (> 120km)",
				raw.AltBaroFt, raw.VerticalRate, distToDest),
			Triggered: true,
		})
	}

	// 8. Condition 2: Heading deviation > 60° sustained far from destination (> 100 km)
	// Must be observed across >= 2 consecutive real observed points with progressing LastContact or timestamp.
	de.mu.Lock()
	if headingDev > 60.0 && distToDest > 100.0 {
		if raw.LastContact > hist.LastContactEpoch || (raw.LastContact == 0 && !now.Equal(hist.LastTimestamp)) {
			hist.DeviationCount++
			hist.LastContactEpoch = raw.LastContact
			if hist.LastDeviationTime.IsZero() {
				hist.LastDeviationTime = now
			}
		}
	} else {
		hist.DeviationCount = 0
		hist.LastDeviationTime = time.Time{}
		if raw.LastContact > hist.LastContactEpoch {
			hist.LastContactEpoch = raw.LastContact
		}
	}
	hist.LastTimestamp = now
	hist.LastAltitude = raw.AltBaroFt
	devCount := hist.DeviationCount
	de.mu.Unlock()

	isSustainedHeadingDeviation := devCount >= 2

	if isSustainedHeadingDeviation {
		analysis.TriggeredCriteria = append(analysis.TriggeredCriteria, models.DiversionCriterion{
			Code: "HEADING_DEVIATION",
			Message: fmt.Sprintf("Heading deviation of %.1f° (> 60°) sustained away from scheduled destination bearing (%.1f°)",
				headingDev, bearingToDest),
			Triggered: true,
		})
	}

	// 9. Condition 3: Descending towards an alternate airport
	var candidateAlt *models.AirportInfo
	if distToDest > 80.0 && raw.AltBaroFt < 18000.0 && raw.VerticalRate < -400 && de.airportDB != nil {
		exclude := []string{}
		if route.Destination != nil {
			exclude = append(exclude, route.Destination.ICAO, route.Destination.IATA)
		}
		if route.Origin != nil {
			exclude = append(exclude, route.Origin.ICAO, route.Origin.IATA)
		}

		cand, candDist, candBearingDiff := de.airportDB.FindCandidateAlternate(raw.Lat, raw.Lon, raw.Track, raw.AltBaroFt, 80.0, exclude...)
		if cand != nil && candDist <= 80.0 && candBearingDiff <= 25.0 {
			candidateAlt = cand
			analysis.CandidateAlternate = candidateAlt
			analysis.TriggeredCriteria = append(analysis.TriggeredCriteria, models.DiversionCriterion{
				Code: "APPROACHING_ALTERNATE_AIRPORT",
				Message: fmt.Sprintf("Descending towards alternate airport %s (%s, %.1fkm away, bearing alignment diff: %.1f°)",
					cand.Name, cand.ICAO, candDist, candBearingDiff),
				Triggered: true,
			})
		}
	}

	// 10. Three-Tier Evaluation:
	// Case A: Extrapolated / estimated point deviation -> Tier "unverified"
	if raw.Source == "estimated" || raw.Source == "stale" {
		if headingDev > 60.0 || isDescendingFar {
			analysis.StatusTier = "unverified"
			analysis.PredictedDiversion = false
			analysis.ProbabilityLevel = "NONE"
			analysis.TriggerReasons = append(analysis.TriggerReasons, "Estimated dead-reckoning trajectory deviation (unverified)")
			if de.onUnverified != nil {
				de.onUnverified(raw.Hex)
			}
		}
		return analysis
	}

	// Squawk 7700 declared emergency with multiple criteria -> CRITICAL emergency diversion
	if raw.Squawk == "7700" && len(analysis.TriggeredCriteria) >= 2 {
		analysis.PredictedDiversion = true
		analysis.StatusTier = "suspected"
		analysis.ProbabilityLevel = "CRITICAL"
		if candidateAlt != nil && candidateAlt.DistanceKm < 15.0 && raw.AltBaroFt < 6000.0 {
			analysis.StatusTier = "confirmed"
		}
		for _, crit := range analysis.TriggeredCriteria {
			analysis.TriggerReasons = append(analysis.TriggerReasons, crit.Message)
		}
		return analysis
	}

	// Case B: Single real point deviation -> Tier "unverified"
	if devCount <= 1 && (headingDev > 60.0 || isDescendingFar) {
		analysis.StatusTier = "unverified"
		analysis.PredictedDiversion = false
		analysis.ProbabilityLevel = "LOW"
		analysis.TriggerReasons = append(analysis.TriggerReasons, "Single point deviation observed (unverified, queued for calibration)")
		if de.onUnverified != nil {
			de.onUnverified(raw.Hex)
		}
		return analysis
	}

	// Case C: No sustained heading deviation or no candidate alternate -> Not a confirmed/suspected diversion
	if !isSustainedHeadingDeviation || candidateAlt == nil {
		analysis.PredictedDiversion = false
		analysis.StatusTier = "none"
		if isEmergencySquawk {
			analysis.ProbabilityLevel = "MODERATE"
		} else {
			analysis.ProbabilityLevel = "NONE"
		}
		return analysis
	}

	// Case D: >= 2 consecutive real points deviating + Candidate alternate identified
	exclude := []string{}
	if route.Destination != nil {
		exclude = append(exclude, route.Destination.ICAO, route.Destination.IATA)
	}
	if route.Origin != nil {
		exclude = append(exclude, route.Origin.ICAO, route.Origin.IATA)
	}
	analysis.TopCandidates = de.airportDB.FindTopCandidateAlternates(raw.Lat, raw.Lon, raw.Track, raw.AltBaroFt, 80.0, exclude...)

	// Check if "confirmed" (distance < 15km, low altitude)
	if candidateAlt.DistanceKm < 15.0 && raw.AltBaroFt < 6000.0 {
		analysis.StatusTier = "confirmed"
		analysis.PredictedDiversion = true
		analysis.ProbabilityLevel = "CRITICAL"
	} else {
		analysis.StatusTier = "suspected"
		analysis.PredictedDiversion = true
		if raw.Squawk == "7700" || len(analysis.TriggeredCriteria) >= 3 {
			analysis.ProbabilityLevel = "HIGH"
		} else {
			analysis.ProbabilityLevel = "MODERATE"
		}
	}

	for _, crit := range analysis.TriggeredCriteria {
		analysis.TriggerReasons = append(analysis.TriggerReasons, crit.Message)
	}

	return analysis
}

func (de *DiversionEngine) checkCorridorDeviation(origICAO, destICAO string, acLat, acLon, origLat, origLon, destLat, destLon float64) (bool, string) {
	routeDist := HaversineDistance(origLat, origLon, destLat, destLon)
	if routeDist < 120.0 {
		return false, ""
	}

	// 1. Try learned historical corridor
	if de.db != nil && origICAO != "" && destICAO != "" {
		corridor, found, err := de.db.GetRouteCorridor(origICAO, destICAO)
		if err == nil && found && corridor.SampleCount >= 5 && len(corridor.Segments) > 0 {
			minSegDist := math.MaxFloat64
			var matchedSeg *database.CorridorSegment
			for _, seg := range corridor.Segments {
				d := HaversineDistance(acLat, acLon, seg.CenterLat, seg.CenterLon)
				if d < minSegDist {
					minSegDist = d
					matchedSeg = &seg
				}
			}
			allowedWidth := 150.0
			if matchedSeg != nil && matchedSeg.WidthKm > 0 {
				allowedWidth = math.Max(matchedSeg.WidthKm*1.5, 120.0)
			}
			if minSegDist > allowedWidth {
				return true, fmt.Sprintf("Deviated from learned corridor (%s-%s, dist to centerline: %.1f km > %.1f km)", origICAO, destICAO, minSegDist, allowedWidth)
			}
			return false, ""
		}
	}

	// 2. Fallback to Great Circle baseline
	xtd := CrossTrackDistance(acLat, acLon, origLat, origLon, destLat, destLon)
	maxAllowedXTD := math.Max(180.0, routeDist*0.25)
	if xtd > maxAllowedXTD {
		return true, fmt.Sprintf("Deviated from great circle path (cross-track dist: %.1f km > %.1f km)", xtd, maxAllowedXTD)
	}
	return false, ""
}

func detectHoldingPattern(points []HistoryPoint) bool {
	if len(points) < 4 {
		return false
	}
	maxDist := 0.0
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			d := HaversineDistance(points[i].Lat, points[i].Lon, points[j].Lat, points[j].Lon)
			if d > maxDist {
				maxDist = d
			}
		}
	}
	// Holding pattern loop must be localized within 20 km
	if maxDist > 20.0 {
		return false
	}

	var totalTurn float64
	for i := 1; i < len(points); i++ {
		prev := points[i-1].Track
		curr := points[i].Track
		diff := curr - prev
		for diff > 180 {
			diff -= 360
		}
		for diff < -180 {
			diff += 360
		}
		totalTurn += diff
	}

	return math.Abs(totalTurn) >= 300.0
}

// CrossTrackDistance calculates perpendicular distance in km from point (lat, lon) to great circle path between (lat1, lon1) and (lat2, lon2).
func CrossTrackDistance(lat, lon, lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	d13 := HaversineDistance(lat1, lon1, lat, lon) / earthRadiusKm
	theta13 := InitialBearing(lat1, lon1, lat, lon) * math.Pi / 180.0
	theta12 := InitialBearing(lat1, lon1, lat2, lon2) * math.Pi / 180.0
	xtd := math.Asin(math.Sin(d13) * math.Sin(theta13-theta12)) * earthRadiusKm
	return math.Abs(xtd)
}

// CleanupStale removes history for aircraft not seen in the last 30 minutes.
func (de *DiversionEngine) CleanupStale(olderThan time.Duration) {
	de.mu.Lock()
	defer de.mu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	for hex, h := range de.history {
		if h.LastTimestamp.Before(cutoff) {
			delete(de.history, hex)
		}
	}
}

// ValidateRoutePlausibility verifies whether a route matches the physical flight state.
// Detects inverted return legs and outdated flight schedules (e.g. ASA286 flying 167° on a LAX->EWR route).
func ValidateRoutePlausibility(raw *models.RawAircraftSnapshot, route *models.RouteInfo) (isPlausible bool, isReversed bool, reason string) {
	if route == nil || route.Origin == nil || route.Destination == nil {
		return false, false, "Route endpoints incomplete"
	}
	origLat := route.Origin.Location.Latitude
	origLon := route.Origin.Location.Longitude
	destLat := route.Destination.Location.Latitude
	destLon := route.Destination.Location.Longitude

	if (origLat == 0 && origLon == 0) || (destLat == 0 && destLon == 0) {
		return false, false, "Zero coordinates for origin or destination"
	}

	distToOrig := HaversineDistance(raw.Lat, raw.Lon, origLat, origLon)
	distToDest := HaversineDistance(raw.Lat, raw.Lon, destLat, destLon)
	routeDist := HaversineDistance(origLat, origLon, destLat, destLon)

	if routeDist < 30.0 {
		return true, false, "Local flight"
	}

	// 1. Geographic corridor ellipse check:
	// If sum of distances exceeds 1.35x route length + 200 km, the aircraft is outside the route's geographic corridor.
	if (distToOrig + distToDest) > (routeDist*1.35 + 200.0) {
		return false, false, fmt.Sprintf("Position outside route corridor (d_orig+d_dest=%.1fkm > max %.1fkm)", distToOrig+distToDest, routeDist*1.35+200.0)
	}

	// 2. Return leg / Inverted route check:
	// E.g., route listed as LAX->EWR, but aircraft is actually flying EWR->LAX (return leg on same callsign).
	bearingToOrig := InitialBearing(raw.Lat, raw.Lon, origLat, origLon)
	bearingToDest := InitialBearing(raw.Lat, raw.Lon, destLat, destLon)

	trackDevToDest := AngularDifference(raw.Track, bearingToDest)
	trackDevToOrig := AngularDifference(raw.Track, bearingToOrig)

	// If aircraft is flying away from destination and directly towards origin (within 35 deg),
	// while in cruise (FL200+, level flight) far from origin (> 80 km), this is an inverted return leg!
	if trackDevToDest > 80.0 && trackDevToOrig < 35.0 && distToOrig > 80.0 && raw.AltBaroFt >= 20000.0 && raw.VerticalRate > -400 {
		return true, true, "Inverted return leg detected"
	}

	// 3. Complete heading contradiction check (e.g. ASA286 heading 167° vs EWR bearing 71°, deviation 96°):
	// If heading deviation is > 80° from bearing to destination while en-route (> 80 km from origin and destination),
	// and track does not align with the route bearing, the route schedule is outdated or mismatched.
	routeBearing := InitialBearing(origLat, origLon, destLat, destLon)
	trackDevRoute := AngularDifference(raw.Track, routeBearing)
	if distToDest > 80.0 && distToOrig > 60.0 && trackDevToDest > 80.0 && trackDevRoute > 80.0 && raw.VerticalRate >= -400 {
		return false, false, fmt.Sprintf("Heading %.1f° conflicts with destination bearing %.1f° and route bearing %.1f° (outdated schedule or callsign reuse)",
			raw.Track, bearingToDest, routeBearing)
	}

	return true, false, "Route plausible"
}
