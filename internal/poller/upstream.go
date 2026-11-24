package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"contrail/internal/config"
	"contrail/internal/database"
	"contrail/internal/models"
)

// GlobalHub represents a geographic center for ADS-B regional aggregation.
type GlobalHub struct {
	Name string
	Lat  float64
	Lon  float64
}

// defaultGlobalHubs covers 69 major international airspace hubs worldwide.
var defaultGlobalHubs = []GlobalHub{
	// East Asia & China
	{"Shanghai", 31.14, 121.80},
	{"Beijing", 40.08, 116.58},
	{"Guangzhou", 23.39, 113.30},
	{"Chengdu", 30.58, 103.95},
	{"Shenzhen", 22.64, 113.81},
	{"Xi'an", 34.45, 108.75},
	{"Hong Kong", 22.31, 113.91},
	{"Taipei", 25.08, 121.23},
	{"Tokyo", 35.76, 140.39},
	{"Osaka", 34.43, 135.24},
	{"Seoul", 37.47, 126.44},

	// Southeast & South Asia
	{"Singapore", 1.36, 103.99},
	{"Bangkok", 13.69, 100.75},
	{"Kuala Lumpur", 2.75, 101.71},
	{"Jakarta", -6.13, 106.66},
	{"Manila", 14.51, 121.02},
	{"Delhi", 28.56, 77.10},
	{"Mumbai", 19.09, 72.87},

	// Middle East
	{"Dubai", 25.25, 55.36},
	{"Doha", 25.27, 51.57},
	{"Riyadh", 24.96, 46.70},
	{"Istanbul", 41.28, 28.75},
	{"Tel Aviv", 32.01, 34.88},
	{"Cairo", 30.12, 31.41},

	// Europe
	{"London", 51.47, -0.45},
	{"Paris", 49.00, 2.55},
	{"Frankfurt", 50.03, 8.57},
	{"Amsterdam", 52.31, 4.76},
	{"Madrid", 40.49, -3.57},
	{"Rome", 41.80, 12.24},
	{"Zurich", 47.46, 8.55},
	{"Munich", 48.35, 11.79},
	{"Vienna", 48.11, 16.56},
	{"Warsaw", 52.17, 20.97},
	{"Copenhagen", 55.62, 12.65},
	{"Stockholm", 59.65, 17.92},
	{"Helsinki", 60.32, 24.96},
	{"Dublin", 53.42, -6.27},
	{"Lisbon", 38.77, -9.13},
	{"Athens", 37.94, 23.94},

	// Africa
	{"Johannesburg", -26.14, 28.25},
	{"Nairobi", -1.32, 36.93},
	{"Addis Ababa", 8.98, 38.80},
	{"Lagos", 6.58, 3.32},
	{"Casablanca", 33.37, -7.59},

	// North America
	{"New York", 40.64, -73.78},
	{"Chicago", 41.97, -87.90},
	{"Atlanta", 33.64, -84.43},
	{"Dallas", 32.90, -97.04},
	{"Los Angeles", 33.94, -118.41},
	{"San Francisco", 37.62, -122.37},
	{"Seattle", 47.45, -122.31},
	{"Miami", 25.80, -80.29},
	{"Denver", 39.86, -104.67},
	{"Toronto", 43.68, -79.63},
	{"Vancouver", 49.19, -123.18},
	{"Honolulu", 21.32, -157.92},
	{"Anchorage", 61.17, -149.99},

	// South America
	{"Sao Paulo", -23.43, -46.47},
	{"Rio de Janeiro", -22.81, -43.25},
	{"Buenos Aires", -34.82, -58.54},
	{"Bogota", 4.70, -74.15},
	{"Santiago", -33.39, -70.79},
	{"Lima", -12.02, -77.11},

	// Oceania
	{"Sydney", -33.94, 151.17},
	{"Melbourne", -37.67, 144.84},
	{"Brisbane", -27.38, 153.12},
	{"Perth", -31.94, 115.97},
	{"Auckland", -37.01, 174.79},
}

// CalculateOpenSkyGlobalInterval computes the uniform budget-pacing interval for the global sub-budget.
// Global calls per day: (dailyLimit * globalShare) / costPerCall.
// For dailyLimit=4000, globalShare=0.65, costPerCall=4 -> 650 calls/day -> 86400 / 650 = ~133s.
func CalculateOpenSkyGlobalInterval(remainingGlobalCredits int, globalDailyBudget int, costPerCall int, minIntervalSec int) time.Duration {
	if costPerCall <= 0 {
		costPerCall = 4
	}
	if minIntervalSec <= 0 {
		minIntervalSec = 100
	}
	if globalDailyBudget <= 0 {
		globalDailyBudget = 2600
	}

	dailyCallsTarget := float64(globalDailyBudget) / float64(costPerCall)
	if dailyCallsTarget <= 0 {
		dailyCallsTarget = 1.0
	}

	baseInterval := 86400.0 / dailyCallsTarget
	if baseInterval < float64(minIntervalSec) {
		baseInterval = float64(minIntervalSec)
	}

	safeThreshold := float64(globalDailyBudget) * 0.20
	finalInterval := baseInterval

	if remainingGlobalCredits <= 0 {
		finalInterval = 900.0 // 15 min maximum backoff
	} else if float64(remainingGlobalCredits) < safeThreshold {
		factor := safeThreshold / float64(remainingGlobalCredits)
		finalInterval = baseInterval * factor
		if finalInterval > 900.0 {
			finalInterval = 900.0
		}
	}

	return time.Duration(math.Round(finalInterval)) * time.Second
}

// CalculateOpenSkyInterval provides backwards compatibility.
func CalculateOpenSkyInterval(remainingCredits int, dailyLimit int, costPerCall int, minIntervalSec int) time.Duration {
	return CalculateOpenSkyGlobalInterval(remainingCredits, dailyLimit, costPerCall, minIntervalSec)
}

// CalculateOpenSkyAreaCost computes credit cost based on OpenSky area pricing rules:
// - Area <= 25 deg^2: 1 credit
// - Area 25 to 100 deg^2: 2 credits
// - Area 100 to 400 deg^2: 3 credits
// - Area > 400 deg^2 or global: 4 credits
func CalculateOpenSkyAreaCost(lamin, lomin, lamax, lomax float64) int {
	area := math.Abs((lamax - lamin) * (lomax - lomin))
	if area <= 0 {
		return 4
	}
	if area <= 25.0 {
		return 1
	}
	if area <= 100.0 {
		return 2
	}
	if area <= 400.0 {
		return 3
	}
	return 4
}

// BuildWeightedHubSequence generates a fair round-robin rotation where daytime hubs (07:00-21:00 local time)
// appear with weight 2 and nighttime hubs appear with weight 1.
func BuildWeightedHubSequence(hubs []GlobalHub, nowUTC time.Time) []GlobalHub {
	utcHour := nowUTC.Hour()
	var daytimeHubs []GlobalHub
	var nightHubs []GlobalHub

	for _, h := range hubs {
		localHour := (utcHour + int(math.Round(h.Lon/15.0))%24 + 24) % 24
		if localHour >= 7 && localHour <= 21 {
			daytimeHubs = append(daytimeHubs, h)
		} else {
			nightHubs = append(nightHubs, h)
		}
	}

	var seq []GlobalHub
	dIdx, nIdx := 0, 0
	for dIdx < len(daytimeHubs) || nIdx < len(nightHubs) {
		for i := 0; i < 2 && dIdx < len(daytimeHubs); i++ {
			seq = append(seq, daytimeHubs[dIdx])
			dIdx++
		}
		if nIdx < len(nightHubs) {
			seq = append(seq, nightHubs[nIdx])
			nIdx++
		}
	}
	for _, h := range daytimeHubs {
		seq = append(seq, h)
	}

	return seq
}

// OpenSkyScheduler runs in an independent goroutine with budget pacing and persistent cooldown.
type OpenSkyScheduler struct {
	cfg         *config.Config
	db          *database.DB
	client      *http.Client
	state       *models.UpstreamState
	mu          sync.RWMutex
	ingestFn    func(snaps []*models.RawAircraftSnapshot, source string)
	oauthToken  string
	tokenExpiry time.Time
	tokenMu     sync.Mutex
}

func NewOpenSkyScheduler(cfg *config.Config, db *database.DB, ingestFn func(snaps []*models.RawAircraftSnapshot, source string)) *OpenSkyScheduler {
	var state *models.UpstreamState
	if db != nil {
		state, _ = db.GetUpstreamState("opensky")
	}
	if state == nil {
		state = &models.UpstreamState{
			Provider:         "opensky",
			Status:           "ok",
			RemainingCredits: 400,
		}
	}

	return &OpenSkyScheduler{
		cfg: cfg,
		db:  db,
		client: &http.Client{
			Timeout: 45 * time.Second,
		},
		state:    state,
		ingestFn: ingestFn,
	}
}

// getBearerToken obtains or refreshes the OpenSky OAuth2 token via Keycloak client_credentials flow.
func (s *OpenSkyScheduler) getBearerToken(ctx context.Context) (string, error) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	// Reuse valid token with 60s buffer
	if s.oauthToken != "" && time.Now().Before(s.tokenExpiry.Add(-60*time.Second)) {
		return s.oauthToken, nil
	}

	if s.cfg.OpenSkyClientID == "" || s.cfg.OpenSkyClientSecret == "" {
		return "", nil
	}

	tokenEndpoint := "https://auth.opensky-network.org/auth/realms/opensky-network/protocol/openid-connect/token"
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", s.cfg.OpenSkyClientID)
	form.Set("client_secret", s.cfg.OpenSkyClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create oauth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oauth token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode oauth token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("oauth token response missing access_token")
	}

	s.oauthToken = tokenResp.AccessToken
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 1800
	}
	s.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	log.Printf("[OpenSkyScheduler] Successfully refreshed OAuth2 Bearer token (expires in %ds)", expiresIn)

	return s.oauthToken, nil
}

func (s *OpenSkyScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	state, err := s.db.GetUpstreamState("opensky")
	if err == nil && state != nil {
		s.state = state
	}

	// If authenticated credentials are configured, verify OAuth2 token and clear any stale anonymous cooldown
	if s.cfg.OpenSkyClientID != "" && s.cfg.OpenSkyClientSecret != "" {
		if s.state.Status == "cooldown" {
			log.Printf("[OpenSkyScheduler] Authenticated credentials configured (%s). Checking token to clear previous anonymous cooldown...", s.cfg.OpenSkyClientID)
			token, err := s.getBearerToken(ctx)
			if err == nil && token != "" {
				s.state.Status = "ok"
				s.state.CooldownUntil = 0
				s.state.NextAllowedAt = time.Now().Unix()
				s.state.ConsecutiveFailures = 0
				if s.state.RemainingCredits < 400 {
					s.state.RemainingCredits = s.cfg.OpenSkyDailyCreditLimit
				}
				_ = s.db.SaveUpstreamState(s.state)
				log.Printf("[OpenSkyScheduler] Previous anonymous cooldown cleared. Scheduler ready in authenticated mode.")
			} else {
				log.Printf("[OpenSkyScheduler] Warning: Failed to verify OAuth2 token at startup: %v", err)
			}
		}
	}
	s.mu.Unlock()

	log.Printf("[OpenSkyScheduler] Initialized. Status: %s, CooldownUntil: %s, NextAllowedAt: %s, RemainingCredits: %d",
		s.state.Status,
		time.Unix(s.state.CooldownUntil, 0).UTC().Format(time.RFC3339),
		time.Unix(s.state.NextAllowedAt, 0).UTC().Format(time.RFC3339),
		s.state.RemainingCredits,
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("[OpenSkyScheduler] Context cancelled, shutting down.")
				return
			default:
			}

			now := time.Now().Unix()

			s.mu.RLock()
			cooldownUntil := s.state.CooldownUntil
			nextAllowedAt := s.state.NextAllowedAt
			s.mu.RUnlock()

			waitSec := int64(0)
			if cooldownUntil > now {
				waitSec = cooldownUntil - now
				log.Printf("[OpenSkyScheduler] 429 Cooldown in effect, sleeping for %d seconds...", waitSec)
			} else if nextAllowedAt > now {
				waitSec = nextAllowedAt - now
			}

			if waitSec > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(waitSec) * time.Second):
				}
			}

			s.fetchGlobalSnapshot(ctx)
		}
	}()
}

func (s *OpenSkyScheduler) fetchGlobalSnapshot(ctx context.Context) {
	start := time.Now()
	nowUnix := start.Unix()

	queryURL := s.cfg.OpenSkyURL
	if s.cfg.OpenSkyBoundingBox != "" {
		parts := strings.Split(s.cfg.OpenSkyBoundingBox, ",")
		if len(parts) == 4 {
			queryURL = fmt.Sprintf("%s?lamin=%s&lomin=%s&lamax=%s&lomax=%s",
				s.cfg.OpenSkyURL, parts[0], parts[1], parts[2], parts[3])
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		s.handleError(0, time.Since(start).Milliseconds(), err.Error())
		return
	}
	req.Header.Set("User-Agent", "Contrail/1.0 (+https://airline.api.veyino.org)")
	req.Header.Set("Accept", "application/json")

	if s.cfg.OpenSkyClientID != "" && s.cfg.OpenSkyClientSecret != "" {
		token, err := s.getBearerToken(ctx)
		if err != nil {
			log.Printf("[OpenSkyScheduler] Warning: Failed to obtain OAuth2 token: %v", err)
		} else if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := s.client.Do(req)
	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		s.handleError(0, durationMs, err.Error())
		return
	}
	defer resp.Body.Close()

	now := time.Now().Unix()
	remHeader := resp.Header.Get("X-Rate-Limit-Remaining")
	if remHeader == "" {
		remHeader = resp.Header.Get("X-RateLimit-Remaining")
	}
	retryHeader := resp.Header.Get("X-Rate-Limit-Retry-After-Seconds")
	if retryHeader == "" {
		retryHeader = resp.Header.Get("Retry-After")
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retrySec, _ := strconv.Atoi(retryHeader)
		if retrySec <= 0 {
			retrySec = 1800
		}
		s.mu.Lock()
		s.state.Status = "cooldown"
		s.state.CooldownUntil = now + int64(retrySec)
		s.state.NextAllowedAt = s.state.CooldownUntil
		s.state.ConsecutiveFailures++
		_ = s.db.SaveUpstreamState(s.state)
		s.mu.Unlock()

		_ = s.db.RecordOpenSkyUsage(now, "global", 0, 429, 0)
		_ = s.db.LogUpstreamFetch("opensky", "global", "rate_limited", 429, 0, durationMs,
			fmt.Sprintf("HTTP 429 Rate limited, cooling down for %d sec", retrySec))
		log.Printf("[OpenSkyScheduler] Received HTTP 429, cooling down for %d seconds", retrySec)
		return
	}

	if resp.StatusCode != http.StatusOK {
		s.handleError(resp.StatusCode, durationMs, fmt.Sprintf("HTTP error: %d", resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.handleError(resp.StatusCode, durationMs, err.Error())
		return
	}

	var data struct {
		Time   int64           `json:"time"`
		States [][]interface{} `json:"states"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		s.handleError(resp.StatusCode, durationMs, "JSON parse error: "+err.Error())
		return
	}

	nowSec := data.Time
	if nowSec == 0 {
		nowSec = time.Now().Unix()
	}

	snapshots := parseOpenSkyStates(data.States, nowSec)

	if s.ingestFn != nil && len(snapshots) > 0 {
		s.ingestFn(snapshots, "opensky")
	}

	// Calculate rolling sub-budget pacing
	globalShare := s.cfg.OpenSkyGlobalShare
	if globalShare <= 0 || globalShare > 1.0 {
		globalShare = 0.65
	}
	globalDailyBudget := int(float64(s.cfg.OpenSkyDailyCreditLimit) * globalShare)

	usedGlobalPast24h, err := s.db.GetOpenSkyUsageInPast24HoursByType(nowUnix, "global")
	if err != nil {
		usedGlobalPast24h = 0
	}
	remCredits := globalDailyBudget - usedGlobalPast24h
	if remCredits < 0 {
		remCredits = 0
	}

	if remHeader != "" {
		if c, err := strconv.Atoi(remHeader); err == nil && c >= 0 {
			remCredits = c
		}
	}

	_ = s.db.RecordOpenSkyUsage(now, "global", s.cfg.OpenSkyCostPerCall, 200, remCredits)

	interval := CalculateOpenSkyGlobalInterval(remCredits, globalDailyBudget, s.cfg.OpenSkyCostPerCall, s.cfg.OpenSkyMinIntervalSec)

	s.mu.Lock()
	s.state.Status = "ok"
	s.state.CooldownUntil = 0
	s.state.ConsecutiveFailures = 0
	s.state.LastSuccessAt = now
	s.state.NextAllowedAt = now + int64(interval.Seconds())
	s.state.RemainingCredits = remCredits
	s.state.LastFetchedCount = len(snapshots)
	_ = s.db.SaveUpstreamState(s.state)
	s.mu.Unlock()

	_ = s.db.LogUpstreamFetch("opensky", "global", "success", 200, len(snapshots), durationMs, "")
	log.Printf("[OpenSkyScheduler] Global snapshot fetched %d states. Credits left: %d. Budget interval: %v (Next: %s)",
		len(snapshots), remCredits, interval, time.Unix(s.state.NextAllowedAt, 0).UTC().Format(time.RFC3339))
}

// CalibrateBoundingBox queries a Tier 1 (Area <= 25 deg^2) regional bounding box on OpenSky for 1 credit.
func (s *OpenSkyScheduler) CalibrateBoundingBox(ctx context.Context, lamin, lomin, lamax, lomax float64) ([]*models.RawAircraftSnapshot, error) {
	s.mu.RLock()
	now := time.Now().Unix()
	if s.state.CooldownUntil > now {
		s.mu.RUnlock()
		return nil, fmt.Errorf("opensky in cooldown until %s", time.Unix(s.state.CooldownUntil, 0).Format(time.RFC3339))
	}
	s.mu.RUnlock()

	// Enforce Tier 1 pricing rule: Area <= 25.0 deg^2 (e.g. 5x5 deg) = 1 credit
	area := (lamax - lamin) * (lomax - lomin)
	if area > 25.0 || area <= 0 {
		centerLat := (lamin + lamax) / 2.0
		centerLon := (lomin + lomax) / 2.0
		lamin = centerLat - 2.5
		lamax = centerLat + 2.5
		lomin = centerLon - 2.5
		lomax = centerLon + 2.5
	}

	start := time.Now()
	queryURL := fmt.Sprintf("%s?lamin=%.4f&lomin=%.4f&lamax=%.4f&lomax=%.4f",
		s.cfg.OpenSkyURL, lamin, lomin, lamax, lomax)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Contrail/1.0 (+https://airline.api.veyino.org)")
	req.Header.Set("Accept", "application/json")
	if s.cfg.OpenSkyClientID != "" && s.cfg.OpenSkyClientSecret != "" {
		token, err := s.getBearerToken(ctx)
		if err != nil {
			log.Printf("[OpenSkyScheduler] Warning: Failed to obtain OAuth2 token for calibration: %v", err)
		} else if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := s.client.Do(req)
	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryHeader := resp.Header.Get("X-Rate-Limit-Retry-After-Seconds")
		if retryHeader == "" {
			retryHeader = resp.Header.Get("Retry-After")
		}
		retrySec, _ := strconv.Atoi(retryHeader)
		if retrySec <= 0 {
			retrySec = 1800
		}
		s.mu.Lock()
		s.state.Status = "cooldown"
		s.state.CooldownUntil = now + int64(retrySec)
		s.state.NextAllowedAt = s.state.CooldownUntil
		_ = s.db.SaveUpstreamState(s.state)
		s.mu.Unlock()

		_ = s.db.RecordOpenSkyUsage(now, "regional", 0, 429, 0)
		_ = s.db.LogUpstreamFetch("opensky", "regional", "rate_limited", 429, 0, durationMs, "HTTP 429 Rate limited")
		return nil, fmt.Errorf("HTTP 429")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Time   int64           `json:"time"`
		States [][]interface{} `json:"states"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	nowSec := data.Time
	if nowSec == 0 {
		nowSec = time.Now().Unix()
	}

	remHeader := resp.Header.Get("X-Rate-Limit-Remaining")
	if remHeader == "" {
		remHeader = resp.Header.Get("X-RateLimit-Remaining")
	}
	if remHeader != "" {
		if c, err := strconv.Atoi(remHeader); err == nil && c >= 0 {
			s.mu.Lock()
			s.state.RemainingCredits = c
			_ = s.db.SaveUpstreamState(s.state)
			s.mu.Unlock()
		}
	}

	cost := CalculateOpenSkyAreaCost(lamin, lomin, lamax, lomax)
	snaps := parseOpenSkyStates(data.States, nowSec)
	_ = s.db.RecordOpenSkyUsage(now, "regional", cost, 200, s.state.RemainingCredits)
	_ = s.db.LogUpstreamFetch("opensky", "regional", "success", 200, len(snaps), durationMs, "")

	if s.ingestFn != nil && len(snaps) > 0 {
		s.ingestFn(snaps, "opensky")
	}

	return snaps, nil
}

func parseOpenSkyStates(states [][]interface{}, nowSec int64) []*models.RawAircraftSnapshot {
	var snapshots []*models.RawAircraftSnapshot
	for _, state := range states {
		if len(state) < 17 {
			continue
		}
		hex, _ := state[0].(string)
		callsign, _ := state[1].(string)
		lastContact := nowSec
		if lc, ok := state[4].(float64); ok && lc > 0 {
			lastContact = int64(lc)
			if lastContact > 10000000000 {
				lastContact /= 1000
			}
		}
		if lastContact < nowSec-300 {
			continue
		}
		lon, okLon := state[5].(float64)
		lat, okLat := state[6].(float64)
		if !okLon || !okLat || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			continue
		}

		altMeters, _ := state[7].(float64)
		altFt := altMeters * 3.28084
		if altFt > 60000 || altFt < -1000 {
			altFt = 0
		}

		onGround, _ := state[8].(bool)
		velocityMps, _ := state[9].(float64)
		gsKts := velocityMps * 1.94384
		if gsKts > 750 || gsKts < 0 {
			gsKts = 0
		}

		track, _ := state[10].(float64)
		vertMps, _ := state[11].(float64)
		vertFpm := vertMps * 196.85

		geoAltMeters, _ := state[13].(float64)
		geoAltFt := geoAltMeters * 3.28084
		squawk, _ := state[14].(string)

		snapshots = append(snapshots, &models.RawAircraftSnapshot{
			Hex:          strings.ToLower(strings.TrimSpace(hex)),
			Flight:       strings.TrimSpace(callsign),
			Squawk:       strings.TrimSpace(squawk),
			Emergency:    "",
			Lat:          lat,
			Lon:          lon,
			AltBaroFt:    altFt,
			AltGeomFt:    geoAltFt,
			GroundSpeed:  gsKts,
			Track:        track,
			VerticalRate: vertFpm,
			OnGround:     onGround,
			LastContact:  lastContact,
			Source:       "opensky",
		})
	}
	return snapshots
}

func (s *OpenSkyScheduler) handleError(statusCode int, durationMs int64, errMsg string) {
	s.mu.Lock()
	s.state.ConsecutiveFailures++
	s.state.Status = "error"
	now := time.Now().Unix()

	backoffSec := int64(s.cfg.OpenSkyMinIntervalSec) * (1 << int64(s.state.ConsecutiveFailures-1))
	if backoffSec > 900 {
		backoffSec = 900
	}
	s.state.NextAllowedAt = now + backoffSec
	remCredits := s.state.RemainingCredits
	_ = s.db.SaveUpstreamState(s.state)
	s.mu.Unlock()

	_ = s.db.RecordOpenSkyUsage(now, "global", 0, statusCode, remCredits)
	_ = s.db.LogUpstreamFetch("opensky", "global", "error", statusCode, 0, durationMs, errMsg)
	log.Printf("[OpenSkyScheduler] Fetch error (code: %d, failures: %d): %s. Backoff: %ds",
		statusCode, s.state.ConsecutiveFailures, errMsg, backoffSec)
}

func (s *OpenSkyScheduler) GetCurrentInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state != nil && s.state.NextAllowedAt > s.state.LastSuccessAt {
		return time.Duration(s.state.NextAllowedAt-s.state.LastSuccessAt) * time.Second
	}
	return time.Duration(s.cfg.OpenSkyMinIntervalSec) * time.Second
}

func (s *OpenSkyScheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().Unix()
	status := "ok"
	if s.state != nil {
		status = s.state.Status
		if s.state.CooldownUntil > now {
			status = "cooldown"
		}
	}

	globalShare := s.cfg.OpenSkyGlobalShare
	if globalShare <= 0 || globalShare > 1.0 {
		globalShare = 0.65
	}
	dailyLimit := s.cfg.OpenSkyDailyCreditLimit
	if dailyLimit <= 0 {
		dailyLimit = 4000
	}
	globalDailyBudget := int(float64(dailyLimit) * globalShare)
	regionalDailyBudget := dailyLimit - globalDailyBudget

	usedGlobal24h := 0
	usedRegional24h := 0
	if s.db != nil {
		usedGlobal24h, _ = s.db.GetOpenSkyUsageInPast24HoursByType(now, "global")
		usedRegional24h, _ = s.db.GetOpenSkyUsageInPast24HoursByType(now, "regional")
	}

	remGlobal := globalDailyBudget - usedGlobal24h
	if remGlobal < 0 {
		remGlobal = 0
	}
	remRegional := regionalDailyBudget - usedRegional24h
	if remRegional < 0 {
		remRegional = 0
	}

	currentInterval := s.GetCurrentInterval()

	res := map[string]interface{}{
		"status": status,
		"global_budget": map[string]interface{}{
			"daily_limit":          globalDailyBudget,
			"used_past_24h":        usedGlobal24h,
			"remaining":            remGlobal,
			"current_interval_sec": int(currentInterval.Seconds()),
		},
		"regional_budget": map[string]interface{}{
			"daily_limit":   regionalDailyBudget,
			"used_past_24h": usedRegional24h,
			"remaining":     remRegional,
		},
		"last_fetched_count": 0,
	}

	if s.state != nil {
		res["last_fetched_count"] = s.state.LastFetchedCount
		if s.state.LastSuccessAt > 0 {
			res["last_success_at"] = time.Unix(s.state.LastSuccessAt, 0).UTC().Format(time.RFC3339)
		}
		if s.state.NextAllowedAt > 0 {
			res["next_request_at"] = time.Unix(s.state.NextAllowedAt, 0).UTC().Format(time.RFC3339)
		}
		if s.state.CooldownUntil > now {
			res["cooldown_until"] = time.Unix(s.state.CooldownUntil, 0).UTC().Format(time.RFC3339)
		}
	}

	return res
}

// HubState tracks per-hub rate limiting.
type HubState struct {
	Hub                 GlobalHub
	LastSuccessAt       time.Time
	ConsecutiveFailures int
	CooldownUntil       time.Time
	LastStatusCode      int
	LastAircraftCount   int
}

// CalibrationTask represents a queued targeted calibration task for adsb.lol.
type CalibrationTask struct {
	Hex         string
	Priority    int // 1: Emergency, 2: Anomaly/Unverified, 3: Descent/Approach, 4: Stale, 5: Regional Hub
	QueuedAt    time.Time
	HubEndpoint string
}

// ADSBMeshScheduler serializes requests to adsb.lol via targeted calibration or optional hub polling.
type ADSBMeshScheduler struct {
	cfg                         *config.Config
	db                          *database.DB
	client                      *http.Client
	allHubs                     []GlobalHub
	hubStates                   map[string]*HubState
	state                       *models.UpstreamState
	sequenceIndex               int
	recent429Timestamps         []time.Time
	globalCooldownFailures      int
	globalCooldownUntil         time.Time
	currentHubInterval          time.Duration
	consecutiveSuccessfulRounds int
	lastRoundAircraft           int
	lastRoundHadErrors          bool

	// Calibration Worker
	calibrationQueue            []CalibrationTask
	hexLastCalibrated           map[string]time.Time
	lastEmergencyCheck          time.Time
	calibrationsCompleted24h    int
	accuracyHook                func(hex string, obsLat, obsLon, obsAltFt, obsTrack float64, lastContact int64)

	mu                          sync.RWMutex
	ingestFn                    func(snaps []*models.RawAircraftSnapshot, source string)
}

func NewADSBMeshScheduler(cfg *config.Config, db *database.DB, ingestFn func(snaps []*models.RawAircraftSnapshot, source string)) *ADSBMeshScheduler {
	hubStates := make(map[string]*HubState)
	for _, h := range defaultGlobalHubs {
		hubStates[h.Name] = &HubState{
			Hub: h,
		}
	}

	var state *models.UpstreamState
	if db != nil {
		state, _ = db.GetUpstreamState("adsb_mesh")
	}
	if state == nil {
		state = &models.UpstreamState{
			Provider: "adsb_mesh",
			Status:   "ok",
		}
	}

	var globalCooldownUntil time.Time
	globalFailures := 0
	if state != nil && state.CooldownUntil > time.Now().Unix() {
		globalCooldownUntil = time.Unix(state.CooldownUntil, 0)
		globalFailures = state.ConsecutiveFailures
	}

	return &ADSBMeshScheduler{
		cfg:                    cfg,
		db:                     db,
		client:                 &http.Client{Timeout: 15 * time.Second},
		allHubs:                defaultGlobalHubs,
		hubStates:              hubStates,
		state:                  state,
		globalCooldownUntil:    globalCooldownUntil,
		globalCooldownFailures: globalFailures,
		currentHubInterval:     3500 * time.Millisecond,
		hexLastCalibrated:      make(map[string]time.Time),
		ingestFn:               ingestFn,
	}
}

// SetAccuracyHook sets the callback invoked when a calibration observation is fetched.
func (s *ADSBMeshScheduler) SetAccuracyHook(fn func(hex string, obsLat, obsLon, obsAltFt, obsTrack float64, lastContact int64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accuracyHook = fn
}

// EnqueueCalibration inserts an aircraft into the targeted calibration priority queue.
func (s *ADSBMeshScheduler) EnqueueCalibration(hex string, priority int) {
	hx := strings.ToLower(strings.TrimSpace(hex))
	if hx == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check deduplication (10 min cooldown)
	cooldownDuration := time.Duration(s.cfg.ADSBLolCalibrationCooldownSec) * time.Second
	if cooldownDuration <= 0 {
		cooldownDuration = 10 * time.Minute
	}
	if last, ok := s.hexLastCalibrated[hx]; ok && time.Since(last) < cooldownDuration {
		return
	}

	// Avoid duplicate entries in queue
	for _, item := range s.calibrationQueue {
		if item.Hex == hx {
			return
		}
	}

	s.calibrationQueue = append(s.calibrationQueue, CalibrationTask{
		Hex:      hx,
		Priority: priority,
		QueuedAt: time.Now(),
	})
}

func (s *ADSBMeshScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	state, err := s.db.GetUpstreamState("adsb_mesh")
	if err == nil && state != nil {
		s.state = state
		if state.CooldownUntil > time.Now().Unix() {
			s.globalCooldownUntil = time.Unix(state.CooldownUntil, 0)
			s.globalCooldownFailures = state.ConsecutiveFailures
		}
	}
	s.mu.Unlock()

	// If Hub polling is disabled (default), run the targeted calibration worker
	if !s.cfg.ADSBLolHubPolling {
		s.startCalibrationWorker(ctx)
		return
	}

	// Otherwise, fallback to hub rotation
	s.startHubPolling(ctx)
}

func (s *ADSBMeshScheduler) startCalibrationWorker(ctx context.Context) {
	log.Printf("[ADSBMeshScheduler] Targeted calibration worker started. Daily limit: %d, min interval: %ds",
		s.cfg.ADSBLolDailyLimit, s.cfg.ADSBLolMinIntervalSec)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			now := time.Now()

			// 1. Check persistent global cooldown
			s.mu.RLock()
			cooldownUntil := s.globalCooldownUntil
			s.mu.RUnlock()

			if now.Before(cooldownUntil) {
				wait := cooldownUntil.Sub(now)
				log.Printf("[ADSBMeshScheduler] In global cooldown, sleeping %v until %s...",
					wait.Truncate(time.Second), cooldownUntil.Format(time.RFC3339))
				select {
				case <-ctx.Done():
					return
				case <-time.After(wait):
				}
				continue
			}

			// 2. Budget pacing calculation
			usedToday, resetAt, err := s.db.GetADSBLolUsageToday(now.Unix())
			if err != nil {
				usedToday = 0
			}

			limit := s.cfg.ADSBLolDailyLimit
			if limit <= 0 {
				limit = 450
			}

			if usedToday >= limit {
				waitSec := resetAt - now.Unix()
				if waitSec > 0 {
					log.Printf("[ADSBMeshScheduler] Daily quota exhausted (%d/%d used today). Waiting %ds until UTC midnight reset.",
						usedToday, limit, waitSec)
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Duration(waitSec) * time.Second):
					}
					continue
				}
			}

			remCalls := limit - usedToday
			remSec := resetAt - now.Unix()
			if remSec < 0 {
				remSec = 86400
			}

			pacingSec := float64(remSec) / float64(remCalls)
			minSec := float64(s.cfg.ADSBLolMinIntervalSec)
			if minSec <= 0 {
				minSec = 30.0
			}
			waitInterval := time.Duration(math.Max(minSec, pacingSec)) * time.Second

			// 3. Pick task to execute
			s.mu.Lock()
			cooldownDuration := time.Duration(s.cfg.ADSBLolCalibrationCooldownSec) * time.Second
			if cooldownDuration <= 0 {
				cooldownDuration = 10 * time.Minute
			}
			for hx, t := range s.hexLastCalibrated {
				if now.Sub(t) > cooldownDuration {
					delete(s.hexLastCalibrated, hx)
				}
			}

			emergInterval := time.Duration(s.cfg.ADSBLolEmergencyIntervalSec) * time.Second
			if emergInterval <= 0 {
				emergInterval = 10 * time.Minute
			}
			isEmergencyDue := now.Sub(s.lastEmergencyCheck) >= emergInterval

			var currentTask *CalibrationTask
			if isEmergencyDue {
				currentTask = &CalibrationTask{
					Hex:      "",
					Priority: 1,
					QueuedAt: now,
				}
				s.lastEmergencyCheck = now
			} else {
				bestIdx := -1
				for i, task := range s.calibrationQueue {
					if last, ok := s.hexLastCalibrated[task.Hex]; ok && now.Sub(last) < cooldownDuration {
						continue
					}
					if bestIdx == -1 || task.Priority < s.calibrationQueue[bestIdx].Priority {
						bestIdx = i
					}
				}

				if bestIdx != -1 {
					task := s.calibrationQueue[bestIdx]
					currentTask = &task
					s.calibrationQueue = append(s.calibrationQueue[:bestIdx], s.calibrationQueue[bestIdx+1:]...)
					s.hexLastCalibrated[task.Hex] = now
				}
			}
			s.mu.Unlock()

			// When calibration queue is empty, utilize the 192s budget slot to poll a weighted regional hub
			if currentTask == nil && len(s.allHubs) > 0 {
				s.mu.Lock()
				hub := s.allHubs[s.sequenceIndex%len(s.allHubs)]
				s.sequenceIndex++
				s.mu.Unlock()
				currentTask = &CalibrationTask{
					Hex:         "",
					Priority:    5,
					QueuedAt:    now,
					HubEndpoint: fmt.Sprintf("/v2/point/%.2f/%.2f/250", hub.Lat, hub.Lon),
				}
			}

			if currentTask == nil {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(minSec) * time.Second):
				}
				continue
			}

			// 4. Perform calibration query
			s.executeCalibration(ctx, currentTask)

			// 5. Enforce budget pacing interval
			select {
			case <-ctx.Done():
				return
			case <-time.After(waitInterval):
			}
		}
	}()
}

func (s *ADSBMeshScheduler) executeCalibration(ctx context.Context, task *CalibrationTask) {
	start := time.Now()
	var endpoint string
	if task.HubEndpoint != "" {
		endpoint = task.HubEndpoint
	} else if task.Priority == 1 || task.Hex == "" {
		endpoint = "/v2/sqk/7700,7600,7500"
	} else {
		endpoint = fmt.Sprintf("/v2/hex/%s", strings.ToLower(task.Hex))
	}

	url := fmt.Sprintf("https://api.adsb.lol%s", endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "Contrail/1.0 (+https://airline.api.veyino.org)")
	req.Header.Set("Accept", "application/json")
	if s.cfg.ADSBLolAPIKey != "" {
		req.Header.Set("x-api-key", s.cfg.ADSBLolAPIKey)
	}

	resp, err := s.client.Do(req)
	durationMs := time.Since(start).Milliseconds()
	nowUnix := time.Now().Unix()

	if err != nil {
		_ = s.db.RecordADSBLolUsage(nowUnix, endpoint, 0)
		_ = s.db.LogUpstreamFetch("adsb_mesh", "calibration", "error", 0, 0, durationMs, err.Error())
		return
	}
	defer resp.Body.Close()

	_ = s.db.RecordADSBLolUsage(nowUnix, endpoint, resp.StatusCode)

	if resp.StatusCode == http.StatusTooManyRequests {
		retryHeader := resp.Header.Get("Retry-After")
		retrySec, _ := strconv.Atoi(retryHeader)
		if retrySec <= 0 {
			retrySec = 1800
		}
		cooldownUntil := time.Now().Add(time.Duration(retrySec) * time.Second)

		s.mu.Lock()
		s.globalCooldownUntil = cooldownUntil
		s.state.Status = "cooldown"
		s.state.CooldownUntil = cooldownUntil.Unix()
		s.state.NextAllowedAt = cooldownUntil.Unix()
		s.state.ConsecutiveFailures++
		_ = s.db.SaveUpstreamState(s.state)
		s.mu.Unlock()

		_ = s.db.LogUpstreamFetch("adsb_mesh", "calibration", "rate_limited", 429, 0, durationMs,
			fmt.Sprintf("HTTP 429 Rate limited, cooling down for %d sec", retrySec))
		log.Printf("[ADSBMeshScheduler] Received 429, cooling down for %d sec (until %s)",
			retrySec, cooldownUntil.Format(time.RFC3339))
		return
	}

	if resp.StatusCode != http.StatusOK {
		_ = s.db.LogUpstreamFetch("adsb_mesh", "calibration", "error", resp.StatusCode, 0, durationMs,
			fmt.Sprintf("HTTP %d", resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var data struct {
		Now float64        `json:"now"`
		AC  []adsbAircraft `json:"ac"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return
	}

	nowSec := int64(data.Now)
	if nowSec > 10000000000 {
		nowSec /= 1000
	} else if nowSec == 0 {
		nowSec = time.Now().Unix()
	}

	var snaps []*models.RawAircraftSnapshot
	for _, ac := range data.AC {
		if ac.Lat == nil || ac.Lon == nil {
			continue
		}
		hexClean := strings.ToLower(strings.TrimSpace(ac.Hex))
		if hexClean == "" {
			continue
		}

		lastContact := nowSec - int64(ac.Seen)
		altBaro := parseNumber(ac.AltBaro)
		altGeom := parseNumber(ac.AltGeom)
		gs := 0.0
		if ac.GS != nil {
			gs = *ac.GS
		}
		track := 0.0
		if ac.Track != nil {
			track = *ac.Track
		}
		vertFpm := 0.0
		if ac.BaroRate != nil {
			vertFpm = *ac.BaroRate
		}

		snap := &models.RawAircraftSnapshot{
			Hex:          hexClean,
			Flight:       strings.TrimSpace(ac.Flight),
			Squawk:       strings.TrimSpace(ac.Squawk),
			Emergency:    ac.Emergency,
			Lat:          *ac.Lat,
			Lon:          *ac.Lon,
			AltBaroFt:    altBaro,
			AltGeomFt:    altGeom,
			GroundSpeed:  gs,
			Track:        track,
			VerticalRate: vertFpm,
			OnGround:     false,
			LastContact:  lastContact,
			Registration: strings.TrimSpace(ac.R),
			Type:         strings.TrimSpace(ac.T),
			Category:     strings.TrimSpace(ac.Category),
			Source:       "adsb_mesh",
		}
		snaps = append(snaps, snap)

		if s.accuracyHook != nil {
			s.accuracyHook(hexClean, *ac.Lat, *ac.Lon, altBaro, track, lastContact)
		}
	}

	s.mu.Lock()
	s.calibrationsCompleted24h++
	s.state.Status = "ok"
	s.state.LastSuccessAt = time.Now().Unix()
	s.state.LastFetchedCount = len(snaps)
	_ = s.db.SaveUpstreamState(s.state)
	s.mu.Unlock()

	if s.ingestFn != nil && len(snaps) > 0 {
		s.ingestFn(snaps, "adsb_mesh")
	}

	_ = s.db.LogUpstreamFetch("adsb_mesh", "calibration", "success", 200, len(snaps), durationMs, "")
}

func (s *ADSBMeshScheduler) startHubPolling(ctx context.Context) {
	log.Printf("[ADSBMeshScheduler] Initialized with %d global hubs (Hub interval: %v).",
		len(s.allHubs), s.currentHubInterval)

	go func() {
		roundAircraft := 0
		roundHadErrors := false

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			now := time.Now()
			s.mu.RLock()
			cooldownUntil := s.globalCooldownUntil
			s.mu.RUnlock()

			if now.Before(cooldownUntil) {
				wait := cooldownUntil.Sub(now)
				select {
				case <-ctx.Done():
					return
				case <-time.After(wait):
				}
				continue
			}

			seq := BuildWeightedHubSequence(s.allHubs, time.Now().UTC())

			if s.sequenceIndex >= len(seq) {
				now = time.Now()
				s.mu.Lock()
				s.lastRoundAircraft = roundAircraft
				s.lastRoundHadErrors = roundHadErrors

				if roundAircraft > 0 {
					s.state.Status = "ok"
					s.state.LastSuccessAt = now.Unix()
					s.state.LastFetchedCount = roundAircraft
					s.globalCooldownFailures = 0
				} else if roundHadErrors {
					if s.state.Status != "cooldown" {
						s.state.Status = "error"
					}
				}

				s.state.NextAllowedAt = now.Unix() + 1
				_ = s.db.SaveUpstreamState(s.state)
				s.sequenceIndex = 0
				s.mu.Unlock()

				roundAircraft = 0
				roundHadErrors = false
				continue
			}

			hub := seq[s.sequenceIndex]
			s.sequenceIndex++

			s.mu.RLock()
			inCooldown := time.Now().Before(s.globalCooldownUntil)
			interval := s.currentHubInterval
			hs := s.hubStates[hub.Name]
			s.mu.RUnlock()

			if inCooldown || (hs != nil && time.Now().Before(hs.CooldownUntil)) {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}

			count, errCode := s.fetchHub(ctx, hub)
			roundAircraft += count
			if errCode != 200 {
				roundHadErrors = true
			}
		}
	}()
}

func (s *ADSBMeshScheduler) triggerGlobalCooldown(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	baseMin := s.cfg.ADSBMeshGlobalCooldownMin
	if baseMin <= 0 {
		baseMin = 15
	}

	mult := 1 << int(math.Min(float64(s.globalCooldownFailures), 3))
	duration := time.Duration(baseMin*mult) * time.Minute
	if duration > 2*time.Hour {
		duration = 2 * time.Hour
	}

	s.globalCooldownFailures++
	s.globalCooldownUntil = now.Add(duration)

	s.state.Status = "cooldown"
	s.state.CooldownUntil = s.globalCooldownUntil.Unix()
	s.state.NextAllowedAt = s.globalCooldownUntil.Unix()
	s.state.ConsecutiveFailures = s.globalCooldownFailures
	_ = s.db.SaveUpstreamState(s.state)

	_ = s.db.LogUpstreamFetch("adsb_mesh", "all", "rate_limited", 429, 0, 0,
		fmt.Sprintf("Global 429 cooldown triggered for %v (until %s)", duration, s.globalCooldownUntil.Format(time.RFC3339)))
}

func (s *ADSBMeshScheduler) fetchHub(ctx context.Context, hub GlobalHub) (int, int) {
	start := time.Now()
	url := fmt.Sprintf("https://api.adsb.lol/v2/point/%.2f/%.2f/250", hub.Lat, hub.Lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, 0
	}
	req.Header.Set("User-Agent", "Contrail/1.0 (+https://airline.api.veyino.org)")
	req.Header.Set("Accept", "application/json")
	if s.cfg.ADSBLolAPIKey != "" {
		req.Header.Set("x-api-key", s.cfg.ADSBLolAPIKey)
	}

	resp, err := s.client.Do(req)
	durationMs := time.Since(start).Milliseconds()

	s.mu.Lock()
	hs := s.hubStates[hub.Name]
	if hs == nil {
		hs = &HubState{Hub: hub}
		s.hubStates[hub.Name] = hs
	}
	s.mu.Unlock()

	if err != nil {
		hs.ConsecutiveFailures++
		hs.LastStatusCode = 0
		_ = s.db.LogUpstreamFetch("adsb_mesh", hub.Name, "error", 0, 0, durationMs, err.Error())
		return 0, 0
	}
	defer resp.Body.Close()

	hs.LastStatusCode = resp.StatusCode

	if resp.StatusCode == http.StatusTooManyRequests {
		now := time.Now()
		s.mu.Lock()
		s.recent429Timestamps = append(s.recent429Timestamps, now)
		cutoff := now.Add(-1 * time.Minute)
		valid := make([]time.Time, 0, len(s.recent429Timestamps))
		for _, t := range s.recent429Timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		s.recent429Timestamps = valid
		shouldGlobalCooldown := len(s.recent429Timestamps) >= 3
		s.mu.Unlock()

		if shouldGlobalCooldown {
			s.triggerGlobalCooldown(now)
			return 0, 429
		}

		retryHeader := resp.Header.Get("Retry-After")
		retrySec, _ := strconv.Atoi(retryHeader)
		if retrySec <= 0 {
			retrySec = 60 * (1 << int(math.Min(float64(hs.ConsecutiveFailures), 4)))
		}
		hs.ConsecutiveFailures++
		hs.CooldownUntil = time.Now().Add(time.Duration(retrySec) * time.Second)

		_ = s.db.LogUpstreamFetch("adsb_mesh", hub.Name, "rate_limited", 429, 0, durationMs,
			fmt.Sprintf("Hub cooling down for %d sec", retrySec))
		return 0, 429
	}

	if resp.StatusCode != http.StatusOK {
		hs.ConsecutiveFailures++
		_ = s.db.LogUpstreamFetch("adsb_mesh", hub.Name, "error", resp.StatusCode, 0, durationMs,
			fmt.Sprintf("HTTP %d", resp.StatusCode))
		return 0, resp.StatusCode
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, resp.StatusCode
	}

	var data struct {
		Now float64        `json:"now"`
		AC  []adsbAircraft `json:"ac"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, resp.StatusCode
	}

	nowSec := int64(data.Now)
	if nowSec > 10000000000 {
		nowSec /= 1000
	} else if nowSec == 0 {
		nowSec = time.Now().Unix()
	}

	var snaps []*models.RawAircraftSnapshot
	for _, ac := range data.AC {
		if ac.Lat == nil || ac.Lon == nil {
			continue
		}
		hexClean := strings.ToLower(strings.TrimSpace(ac.Hex))
		if hexClean == "" {
			continue
		}

		lastContact := nowSec - int64(ac.Seen)
		altBaro := parseNumber(ac.AltBaro)
		altGeom := parseNumber(ac.AltGeom)
		gs := 0.0
		if ac.GS != nil {
			gs = *ac.GS
		}
		track := 0.0
		if ac.Track != nil {
			track = *ac.Track
		}
		vertFpm := 0.0
		if ac.BaroRate != nil {
			vertFpm = *ac.BaroRate
		}

		snaps = append(snaps, &models.RawAircraftSnapshot{
			Hex:          hexClean,
			Flight:       strings.TrimSpace(ac.Flight),
			Squawk:       strings.TrimSpace(ac.Squawk),
			Emergency:    ac.Emergency,
			Lat:          *ac.Lat,
			Lon:          *ac.Lon,
			AltBaroFt:    altBaro,
			AltGeomFt:    altGeom,
			GroundSpeed:  gs,
			Track:        track,
			VerticalRate: vertFpm,
			OnGround:     false,
			LastContact:  lastContact,
			Registration: strings.TrimSpace(ac.R),
			Type:         strings.TrimSpace(ac.T),
			Category:     strings.TrimSpace(ac.Category),
			Source:       "adsb_mesh",
		})
	}

	hs.ConsecutiveFailures = 0
	hs.LastSuccessAt = time.Now()
	hs.LastAircraftCount = len(snaps)

	if s.ingestFn != nil && len(snaps) > 0 {
		s.ingestFn(snaps, "adsb_mesh")
	}

	_ = s.db.LogUpstreamFetch("adsb_mesh", hub.Name, "success", 200, len(snaps), durationMs, "")
	return len(snaps), 200
}

func (s *ADSBMeshScheduler) GetCurrentInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.cfg.ADSBLolHubPolling {
		minSec := float64(s.cfg.ADSBLolMinIntervalSec)
		if minSec <= 0 {
			minSec = 30.0
		}
		return time.Duration(minSec) * time.Second
	}
	hubs := len(s.hubStates)
	if hubs == 0 {
		hubs = len(s.allHubs)
	}
	interval := s.currentHubInterval
	if interval <= 0 {
		interval = 3500 * time.Millisecond
	}
	return interval * time.Duration(hubs)
}

func (s *ADSBMeshScheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	nowUnix := now.Unix()

	limit := s.cfg.ADSBLolDailyLimit
	if limit <= 0 {
		limit = 450
	}

	usedToday := 0
	var resetAt int64
	if s.db != nil {
		usedToday, resetAt, _ = s.db.GetADSBLolUsageToday(nowUnix)
	}
	remToday := limit - usedToday
	if remToday < 0 {
		remToday = 0
	}

	queueLen := len(s.calibrationQueue)
	cal24h := s.calibrationsCompleted24h
	if s.db != nil {
		if c, err := s.db.GetADSBLolUsageInPast24Hours(nowUnix); err == nil && c > cal24h {
			cal24h = c
		}
	}

	totalHubs := len(s.hubStates)
	activeHubs := 0
	cooldownHubs := 0
	inGlobalCooldown := s.globalCooldownUntil.After(now) || (s.state != nil && s.state.CooldownUntil > nowUnix)
	if inGlobalCooldown {
		activeHubs = 0
		cooldownHubs = totalHubs
	} else {
		for _, hs := range s.hubStates {
			if now.Before(hs.CooldownUntil) {
				cooldownHubs++
			} else {
				activeHubs++
			}
		}
	}

	status := "ok"
	if s.state != nil {
		status = s.state.Status
	}
	if inGlobalCooldown || (totalHubs > 0 && activeHubs == 0) {
		status = "cooldown"
	}

	res := map[string]interface{}{
		"status":                     status,
		"total_hubs":                 totalHubs,
		"active_hubs":                activeHubs,
		"cooldown_hubs":              cooldownHubs,
		"hub_polling_enabled":        s.cfg.ADSBLolHubPolling,
		"daily_limit":                limit,
		"used_today":                 usedToday,
		"remaining_today":            remToday,
		"window_reset_at":            time.Unix(resetAt, 0).UTC().Format(time.RFC3339),
		"calibration_queue_len":      queueLen,
		"calibrations_completed_24h": cal24h,
		"last_fetched_count":         0,
	}

	if s.state != nil {
		res["last_fetched_count"] = s.state.LastFetchedCount
		if s.state.LastSuccessAt > 0 {
			res["last_success_at"] = time.Unix(s.state.LastSuccessAt, 0).UTC().Format(time.RFC3339)
		}
	}

	cooldownTime := s.globalCooldownUntil
	if s.state != nil && s.state.CooldownUntil > cooldownTime.Unix() {
		cooldownTime = time.Unix(s.state.CooldownUntil, 0)
	}

	if cooldownTime.After(now) {
		res["cooldown_until"] = cooldownTime.UTC().Format(time.RFC3339)
		res["next_request_at"] = cooldownTime.UTC().Format(time.RFC3339)
	} else if s.state != nil && s.state.NextAllowedAt > nowUnix {
		res["next_request_at"] = time.Unix(s.state.NextAllowedAt, 0).UTC().Format(time.RFC3339)
	}

	return res
}

type adsbAircraft struct {
	Hex       string      `json:"hex"`
	Flight    string      `json:"flight"`
	Squawk    string      `json:"squawk"`
	Emergency string      `json:"emergency"`
	Lat       *float64    `json:"lat"`
	Lon       *float64    `json:"lon"`
	AltBaro   interface{} `json:"alt_baro"`
	AltGeom   interface{} `json:"alt_geom"`
	GS        *float64    `json:"gs"`
	Track     *float64    `json:"track"`
	BaroRate  *float64    `json:"baro_rate"`
	Category  string      `json:"category"`
	Desc      string      `json:"desc"`
	R         string      `json:"r"`
	T         string      `json:"t"`
	Seen      float64     `json:"seen"`
}

func parseNumber(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0.0
	}
}
