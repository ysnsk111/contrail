package poller

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"contrail/internal/database"
	"contrail/internal/models"
	"golang.org/x/time/rate"
)

//go:embed airlines.json
var embeddedAirlinesJSON []byte

var commercialCallsignRegex = regexp.MustCompile(`^[A-Z]{3}[0-9]{1,4}[A-Z]?$`)

// RouteEnricher enriches flights with IATA numbers, airlines, and origin/destination airports using adsbdb.
type RouteEnricher struct {
	db          *database.DB
	httpClient  *http.Client
	baseURL     string
	limiter     *rate.Limiter
	cacheTTL    time.Duration
	negCacheTTL time.Duration
	highQueue   chan string
	lowQueue    chan string
	queueSet    map[string]bool
	queueSetMu  sync.Mutex
	memCache    sync.Map // callsign -> *database.CachedRoute
	airlines    map[string]models.AirlineInfo
}

// NewRouteEnricher creates a new RouteEnricher with token bucket rate limiting and embedded airline database.
func NewRouteEnricher(db *database.DB, baseURL string, rateLimitPerMin int, cacheTTLHours int) *RouteEnricher {
	if rateLimitPerMin <= 0 || rateLimitPerMin > 512 {
		rateLimitPerMin = 400 // Safe rate: 400 req/min (< 512 limit)
	}
	if cacheTTLHours <= 0 {
		cacheTTLHours = 24
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(rateLimitPerMin)), 10)

	enricher := &RouteEnricher{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:     strings.TrimSuffix(baseURL, "/") + "/",
		limiter:     limiter,
		cacheTTL:    time.Duration(cacheTTLHours) * time.Hour,
		negCacheTTL: 6 * time.Hour, // Unknown callsigns cached for 6h
		highQueue:   make(chan string, 20000),
		lowQueue:    make(chan string, 10000),
		queueSet:    make(map[string]bool),
		airlines:    make(map[string]models.AirlineInfo),
	}

	if len(embeddedAirlinesJSON) > 0 {
		var parsed map[string]models.AirlineInfo
		if err := json.Unmarshal(embeddedAirlinesJSON, &parsed); err == nil {
			enricher.airlines = parsed
		} else {
			log.Printf("[Enricher] Warning: Failed to parse embedded airlines.json: %v", err)
		}
	}

	return enricher
}

// StartWorker launches concurrent background consumer workers.
func (re *RouteEnricher) StartWorker(ctx context.Context, workers int) {
	if workers <= 0 {
		workers = 5
	}
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Priority select: check high-priority queue first
				var callsign string
				var ok bool

				select {
				case callsign, ok = <-re.highQueue:
					if !ok {
						return
					}
				default:
					select {
					case callsign, ok = <-re.highQueue:
						if !ok {
							return
						}
					case callsign, ok = <-re.lowQueue:
						if !ok {
							return
						}
					case <-ctx.Done():
						return
					}
				}

				re.queueSetMu.Lock()
				delete(re.queueSet, callsign)
				re.queueSetMu.Unlock()

				// Double-check cache before hitting external API
				if re.isCached(callsign) {
					continue
				}

				// Wait on token bucket limiter (<= 400 req/min)
				if err := re.limiter.Wait(ctx); err != nil {
					return
				}

				re.fetchAndCache(ctx, callsign)
			}
		}()
	}
}

// MatchAirlineAndIATA matches airline details and computes derived IATA callsign.
// If callsign does not match an airline or is non-commercial, returns nil, nil.
func (re *RouteEnricher) MatchAirlineAndIATA(callsign string) (*models.AirlineInfo, *string) {
	trimmed := strings.ToUpper(strings.TrimSpace(callsign))
	if len(trimmed) < 4 {
		return nil, nil
	}

	prefix := trimmed[:3]
	airline, exists := re.airlines[prefix]
	if !exists {
		return nil, nil
	}

	airlineCopy := airline

	// If the rest of the callsign is a valid flight number suffix (digits optionally ending in letter)
	suffix := trimmed[3:]
	if len(suffix) > 0 && isNumericFlightSuffix(suffix) {
		if airline.IATA != "" {
			iataStr := airline.IATA + suffix
			return &airlineCopy, &iataStr
		}
	}

	return &airlineCopy, nil
}

func isNumericFlightSuffix(s string) bool {
	// e.g. "123", "3408", "5101", "12A"
	first := s[0]
	return first >= '0' && first <= '9'
}

func (re *RouteEnricher) isCommercialFormat(callsign string) bool {
	return commercialCallsignRegex.MatchString(callsign)
}

// GetRoute returns the cached route, airline, and IATA callsign.
// If route is missing, queues callsign for background enrichment while returning matched airline metadata.
func (re *RouteEnricher) GetRoute(callsign string) (*models.RouteInfo, *models.AirlineInfo, *string) {
	trimmed := strings.ToUpper(strings.TrimSpace(callsign))
	if trimmed == "" || len(trimmed) < 3 {
		return nil, nil, nil
	}

	// 1. Fast in-memory check
	if val, ok := re.memCache.Load(trimmed); ok {
		cached := val.(*database.CachedRoute)
		if cached.ExpiresAt.After(time.Now()) {
			if cached.Status == "found" {
				var iataPtr *string
				if cached.IATA != "" {
					iataCopy := cached.IATA
					iataPtr = &iataCopy
				} else {
					_, iataPtr = re.MatchAirlineAndIATA(trimmed)
				}
				return cached.Route, cached.Airline, iataPtr
			}
			// Status "unknown" cached: do not query adsbdb again during negative cache TTL
			matchedAir, matchedIATA := re.MatchAirlineAndIATA(trimmed)
			return nil, matchedAir, matchedIATA
		}
	}

	// 2. Database check
	cached, found, err := re.db.GetCachedCallsign(trimmed)
	if err == nil && found {
		re.memCache.Store(trimmed, cached)
		if cached.Status == "found" {
			var iataPtr *string
			if cached.IATA != "" {
				iataCopy := cached.IATA
				iataPtr = &iataCopy
			} else {
				_, iataPtr = re.MatchAirlineAndIATA(trimmed)
			}
			return cached.Route, cached.Airline, iataPtr
		}
		matchedAir, matchedIATA := re.MatchAirlineAndIATA(trimmed)
		return nil, matchedAir, matchedIATA
	}

	// 3. Not in cache -> queue for rate-limited fetching
	re.queueCallsign(trimmed)

	// Return local airline matching immediately so metadata is populated right away
	matchedAir, matchedIATA := re.MatchAirlineAndIATA(trimmed)
	return nil, matchedAir, matchedIATA
}

func (re *RouteEnricher) isCached(callsign string) bool {
	if val, ok := re.memCache.Load(callsign); ok {
		cached := val.(*database.CachedRoute)
		if cached.ExpiresAt.After(time.Now()) {
			return true
		}
	}
	cached, found, err := re.db.GetCachedCallsign(callsign)
	if err == nil && found {
		re.memCache.Store(callsign, cached)
		return true
	}
	return false
}

func (re *RouteEnricher) queueCallsign(callsign string) {
	re.queueSetMu.Lock()
	defer re.queueSetMu.Unlock()

	if re.queueSet[callsign] {
		return
	}

	if re.isCommercialFormat(callsign) {
		select {
		case re.highQueue <- callsign:
			re.queueSet[callsign] = true
		default:
		}
	} else {
		select {
		case re.lowQueue <- callsign:
			re.queueSet[callsign] = true
		default:
		}
	}
}

type adsbDbResponse struct {
	Response struct {
		FlightRoute struct {
			Callsign     string `json:"callsign"`
			CallsignICAO string `json:"callsign_icao"`
			CallsignIATA string `json:"callsign_iata"`
			Airline      struct {
				Name       string `json:"name"`
				ICAO       string `json:"icao"`
				IATA       string `json:"iata"`
				Country    string `json:"country"`
				CountryISO string `json:"country_iso"`
				Callsign   string `json:"callsign"`
			} `json:"airline"`
			Origin struct {
				CountryISOName string  `json:"country_iso_name"`
				CountryName    string  `json:"country_name"`
				Elevation      int     `json:"elevation"`
				IATACode       string  `json:"iata_code"`
				ICAOCode       string  `json:"icao_code"`
				Latitude       float64 `json:"latitude"`
				Longitude      float64 `json:"longitude"`
				Municipality   string  `json:"municipality"`
				Name           string  `json:"name"`
			} `json:"origin"`
			Destination struct {
				CountryISOName string  `json:"country_iso_name"`
				CountryName    string  `json:"country_name"`
				Elevation      int     `json:"elevation"`
				IATACode       string  `json:"iata_code"`
				ICAOCode       string  `json:"icao_code"`
				Latitude       float64 `json:"latitude"`
				Longitude      float64 `json:"longitude"`
				Municipality   string  `json:"municipality"`
				Name           string  `json:"name"`
			} `json:"destination"`
		} `json:"flightroute"`
	} `json:"response"`
}

func (re *RouteEnricher) fetchAndCache(ctx context.Context, callsign string) {
	url := fmt.Sprintf("%s%s", re.baseURL, callsign)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "Contrail/1.0 (+https://airline.api.veyino.org)")

	resp, err := re.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		var apiRes adsbDbResponse
		if err := json.Unmarshal(body, &apiRes); err != nil {
			return
		}

		fr := apiRes.Response.FlightRoute
		if fr.Origin.ICAOCode != "" || fr.Destination.ICAOCode != "" {
			airline := &models.AirlineInfo{
				Name:    fr.Airline.Name,
				ICAO:    fr.Airline.ICAO,
				IATA:    fr.Airline.IATA,
				Country: fr.Airline.Country,
			}

			originAirport := &models.AirportInfo{
				ICAO:    fr.Origin.ICAOCode,
				IATA:    fr.Origin.IATACode,
				Name:    fr.Origin.Name,
				City:    fr.Origin.Municipality,
				Country: fr.Origin.CountryName,
				Location: models.AirportLocation{
					Latitude:    fr.Origin.Latitude,
					Longitude:   fr.Origin.Longitude,
					ElevationFt: fr.Origin.Elevation,
				},
			}

			destAirport := &models.AirportInfo{
				ICAO:    fr.Destination.ICAOCode,
				IATA:    fr.Destination.IATACode,
				Name:    fr.Destination.Name,
				City:    fr.Destination.Municipality,
				Country: fr.Destination.CountryName,
				Location: models.AirportLocation{
					Latitude:    fr.Destination.Latitude,
					Longitude:   fr.Destination.Longitude,
					ElevationFt: fr.Destination.Elevation,
				},
			}

			origCode := fr.Origin.IATACode
			if origCode == "" {
				origCode = fr.Origin.ICAOCode
			}
			destCode := fr.Destination.IATACode
			if destCode == "" {
				destCode = fr.Destination.ICAOCode
			}

			route := &models.RouteInfo{
				Summary:     fmt.Sprintf("%s -> %s", origCode, destCode),
				Origin:      originAirport,
				Destination: destAirport,
			}

			iataCode := fr.CallsignIATA
			if iataCode == "" {
				_, matchedIATA := re.MatchAirlineAndIATA(callsign)
				if matchedIATA != nil {
					iataCode = *matchedIATA
				}
			}

			// Store in DB & memCache
			re.db.SetCachedCallsign(callsign, iataCode, airline, route, "found", re.cacheTTL)
			re.memCache.Store(callsign, &database.CachedRoute{
				Callsign:  callsign,
				Status:    "found",
				Route:     route,
				Airline:   airline,
				IATA:      iataCode,
				CachedAt:  time.Now(),
				ExpiresAt: time.Now().Add(re.cacheTTL),
			})
			return
		}
	}

	// 400, 404, or non-commercial/charter: mark as "unknown" to avoid re-querying for 6h (negCacheTTL)
	re.db.SetCachedCallsign(callsign, "", nil, nil, "unknown", re.negCacheTTL)
	re.memCache.Store(callsign, &database.CachedRoute{
		Callsign:  callsign,
		Status:    "unknown",
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(re.negCacheTTL),
	})
}

// FetchAdsbLolRouteset demonstrates an optional supplementary fallback route resolution
// using adsb.lol's unmetered /api/0/routeset endpoint.
func (re *RouteEnricher) FetchAdsbLolRouteset(ctx context.Context, callsign string) (*models.RouteInfo, error) {
	clean := strings.ToUpper(strings.TrimSpace(callsign))
	if clean == "" {
		return nil, fmt.Errorf("empty callsign")
	}

	url := fmt.Sprintf("https://api.adsb.lol/api/0/routeset/%s", clean)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Contrail/1.0 (+https://airline.api.veyino.org)")
	req.Header.Set("Accept", "application/json")

	resp, err := re.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("routeset HTTP %d", resp.StatusCode)
	}

	var data struct {
		Callsign string `json:"callsign"`
		Origin   string `json:"origin"`
		Dest     string `json:"destination"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Origin == "" || data.Dest == "" {
		return nil, fmt.Errorf("incomplete route returned from routeset")
	}

	return &models.RouteInfo{
		Summary: fmt.Sprintf("%s -> %s", data.Origin, data.Dest),
		Origin: &models.AirportInfo{
			ICAO: data.Origin,
		},
		Destination: &models.AirportInfo{
			ICAO: data.Dest,
		},
	}, nil
}
