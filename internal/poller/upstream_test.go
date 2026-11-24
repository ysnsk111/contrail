package poller

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"contrail/internal/config"
	"contrail/internal/database"
	"contrail/internal/models"
)

func TestCalculateOpenSkyInterval(t *testing.T) {
	// Daily limit 4000, costPerCall 4, minInterval 100s
	// Base interval = 86400 / (4000 * 0.9 / 4) = 86400 / 900 = 96s -> clamped to 100s

	// 1. Ample credits (4000): interval = 100s
	interval := CalculateOpenSkyInterval(4000, 4000, 4, 100)
	if interval != 100*time.Second {
		t.Errorf("Expected 100s for ample credits, got %v", interval)
	}

	// 2. Normal credits (1200): interval = 100s
	interval = CalculateOpenSkyInterval(1200, 4000, 4, 100)
	if interval != 100*time.Second {
		t.Errorf("Expected 100s for normal credits, got %v", interval)
	}

	// 3. Below safe threshold (800 credits = 20%): at 400 credits, slows down by 800/400 = 2.0x -> 200s
	interval = CalculateOpenSkyInterval(400, 4000, 4, 100)
	if interval != 200*time.Second {
		t.Errorf("Expected 200s for 400 credits, got %v", interval)
	}

	// 4. Depleted credits (0): maximum backoff = 900s
	interval = CalculateOpenSkyInterval(0, 4000, 4, 100)
	if interval != 900*time.Second {
		t.Errorf("Expected 900s for 0 credits, got %v", interval)
	}
}

func TestOpenSkyUsageRolling24h(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_opensky_usage.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create db: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()

	// 1. Record usage from 2 hours ago (4 credits)
	if err := db.RecordOpenSkyUsage(now-7200, "global", 4, 200, 3996); err != nil {
		t.Fatalf("RecordOpenSkyUsage failed: %v", err)
	}
	// 2. Record usage from 10 hours ago (4 credits)
	if err := db.RecordOpenSkyUsage(now-36000, "global", 4, 200, 3992); err != nil {
		t.Fatalf("RecordOpenSkyUsage failed: %v", err)
	}
	// 3. Record usage from 25 hours ago (>24h window, 8 credits)
	if err := db.RecordOpenSkyUsage(now-90000, "global", 8, 200, 3984); err != nil {
		t.Fatalf("RecordOpenSkyUsage failed: %v", err)
	}

	// Sum past 24h should only include the first two (4 + 4 = 8)
	used24h, err := db.GetOpenSkyUsageInPast24Hours(now)
	if err != nil {
		t.Fatalf("GetOpenSkyUsageInPast24Hours failed: %v", err)
	}
	if used24h != 8 {
		t.Errorf("Expected 8 credits used in past 24 hours, got %d", used24h)
	}
}

func TestUpstreamStatePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_upstream.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test db: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	initialState := &models.UpstreamState{
		Provider:            "opensky",
		LastSuccessAt:       now - 120,
		NextAllowedAt:       now + 300,
		RemainingCredits:    350,
		ConsecutiveFailures: 2,
		CooldownUntil:       now + 600,
		Status:              "cooldown",
		LastFetchedCount:    1234,
	}

	if err := db.SaveUpstreamState(initialState); err != nil {
		t.Fatalf("Failed to save upstream state: %v", err)
	}

	// Read state back
	loaded, err := db.GetUpstreamState("opensky")
	if err != nil {
		t.Fatalf("Failed to get upstream state: %v", err)
	}
	if loaded == nil {
		t.Fatalf("Expected non-nil upstream state")
	}

	if loaded.Provider != "opensky" || loaded.RemainingCredits != 350 || loaded.ConsecutiveFailures != 2 || loaded.CooldownUntil != now+600 {
		t.Errorf("Loaded state mismatch: %+v", loaded)
	}

	// Verify GetAllUpstreamStates
	allStates, err := db.GetAllUpstreamStates()
	if err != nil {
		t.Fatalf("Failed to get all states: %v", err)
	}
	if len(allStates) != 1 || allStates[0].Provider != "opensky" {
		t.Errorf("Unexpected all states result: %+v", allStates)
	}
}

func TestADSBMeshHubIsolation(t *testing.T) {
	cfg := &config.Config{
		ADSBMeshRateLimitRPS: 100, // fast for testing
	}
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mesh.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create db: %v", err)
	}
	defer db.Close()

	mesh := NewADSBMeshScheduler(cfg, db, nil)
	now := time.Now()

	// Simulate 429 on Beijing hub
	mesh.mu.Lock()
	beijingState := mesh.hubStates["Beijing"]
	if beijingState == nil {
		mesh.mu.Unlock()
		t.Fatalf("Beijing hub state not found in default hubs")
	}
	// Simulate isolated backoff for Beijing
	beijingState.ConsecutiveFailures = 1
	beijingState.CooldownUntil = now.Add(60 * time.Second)
	beijingState.LastStatusCode = 429
	mesh.mu.Unlock()

	status := mesh.GetStatus()
	if status["cooldown_hubs"].(int) != 1 {
		t.Errorf("Expected 1 cooldown hub, got %v", status["cooldown_hubs"])
	}
	if status["active_hubs"].(int) != len(mesh.hubStates)-1 {
		t.Errorf("Expected %d active hubs, got %v", len(mesh.hubStates)-1, status["active_hubs"])
	}

	// Verify other hub (e.g. London) is unaffected
	mesh.mu.RLock()
	londonState := mesh.hubStates["London"]
	mesh.mu.RUnlock()
	if londonState == nil || !now.After(londonState.CooldownUntil) {
		t.Errorf("Expected London to be active and not in cooldown")
	}
}

func TestIngestBatchMergeRules(t *testing.T) {
	cfg := &config.Config{
		OpenSkyMinIntervalSec:   100,
		OpenSkyCostPerCall:      4,
		ADSBMeshRateLimitRPS:    1.0,
		DeadReckoningMaxAgeSec:  180,
		AircraftPurgeTimeoutSec: 600,
		SourcePriority:          []string{"opensky", "adsb_mesh"},
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_merge.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test db: %v", err)
	}
	defer db.Close()

	p := NewPoller(cfg, db)

	now := time.Now().Unix()

	// 1. Ingest flight A from adsb_mesh at t = now - 20
	snapA1 := &models.RawAircraftSnapshot{
		Hex:          "a11111",
		Flight:       "TEST01",
		Lat:          39.9,
		Lon:          116.4,
		LastContact:  now - 20,
		AltBaroFt:    30000,
		GroundSpeed:  400,
		Track:        90,
		VerticalRate: 0,
	}
	p.IngestBatch([]*models.RawAircraftSnapshot{snapA1}, "adsb_mesh")

	p.mu.RLock()
	flightA := p.liveCache["a11111"]
	p.mu.RUnlock()
	if flightA == nil || flightA.Metadata.UpstreamSource != "adsb_mesh" || flightA.Telemetry.LastObservedAt != now-20 {
		t.Fatalf("Flight A not ingested correctly: %+v", flightA)
	}

	// 2. Ingest older observation for flight A from opensky at t = now - 50 -> must NOT overwrite
	snapAOlder := &models.RawAircraftSnapshot{
		Hex:          "a11111",
		Flight:       "TEST01",
		Lat:          38.0,
		Lon:          115.0,
		LastContact:  now - 50,
		AltBaroFt:    25000,
		GroundSpeed:  350,
		Track:        80,
		VerticalRate: 0,
	}
	p.IngestBatch([]*models.RawAircraftSnapshot{snapAOlder}, "opensky")

	p.mu.RLock()
	flightA = p.liveCache["a11111"]
	p.mu.RUnlock()
	if flightA.Telemetry.LastObservedAt != now-20 || flightA.Metadata.UpstreamSource != "adsb_mesh" {
		t.Errorf("Older observation overwrote newer observation! Got source: %s, last_contact: %d",
			flightA.Metadata.UpstreamSource, flightA.Telemetry.LastObservedAt)
	}

	// 3. Ingest newer observation for flight A from opensky at t = now - 5 -> MUST overwrite
	snapANewer := &models.RawAircraftSnapshot{
		Hex:          "a11111",
		Flight:       "TEST01",
		Lat:          40.0,
		Lon:          116.8,
		LastContact:  now - 5,
		AltBaroFt:    32000,
		GroundSpeed:  420,
		Track:        95,
		VerticalRate: 100,
	}
	p.IngestBatch([]*models.RawAircraftSnapshot{snapANewer}, "opensky")

	p.mu.RLock()
	flightA = p.liveCache["a11111"]
	p.mu.RUnlock()
	if flightA.Telemetry.LastObservedAt != now-5 || flightA.Metadata.UpstreamSource != "opensky" {
		t.Errorf("Newer observation did not overwrite! Got source: %s, last_contact: %d",
			flightA.Metadata.UpstreamSource, flightA.Telemetry.LastObservedAt)
	}

	// 4. Ingest flight B (partial batch). Flight A MUST NOT be deleted.
	snapB := &models.RawAircraftSnapshot{
		Hex:          "b22222",
		Flight:       "TEST02",
		Lat:          31.2,
		Lon:          121.5,
		LastContact:  now - 10,
		AltBaroFt:    20000,
		GroundSpeed:  380,
		Track:        180,
		VerticalRate: 0,
	}
	p.IngestBatch([]*models.RawAircraftSnapshot{snapB}, "adsb_mesh")

	p.mu.RLock()
	flightACheck := p.liveCache["a11111"]
	flightBCheck := p.liveCache["b22222"]
	p.mu.RUnlock()

	if flightACheck == nil {
		t.Errorf("Flight A was deleted when partial batch for Flight B was ingested!")
	}
	if flightBCheck == nil {
		t.Errorf("Flight B was not ingested!")
	}

	// 5. Test tie-breaking by source priority when last_contact is equal
	// opensky has higher priority than adsb_mesh
	snapTieMesh := &models.RawAircraftSnapshot{
		Hex:          "c33333",
		Flight:       "TEST03",
		Lat:          51.5,
		Lon:          -0.1,
		LastContact:  now - 15,
		AltBaroFt:    15000,
		GroundSpeed:  300,
	}
	p.IngestBatch([]*models.RawAircraftSnapshot{snapTieMesh}, "adsb_mesh")

	snapTieOpenSky := &models.RawAircraftSnapshot{
		Hex:          "c33333",
		Flight:       "TEST03",
		Lat:          51.51,
		Lon:          -0.11,
		LastContact:  now - 15, // identical timestamp
		AltBaroFt:    15050,
		GroundSpeed:  305,
	}
	p.IngestBatch([]*models.RawAircraftSnapshot{snapTieOpenSky}, "opensky")

	p.mu.RLock()
	flightC := p.liveCache["c33333"]
	p.mu.RUnlock()
	if flightC == nil || flightC.Metadata.UpstreamSource != "opensky" {
		t.Errorf("Expected higher priority source 'opensky' to win tie-break, got %v", flightC.Metadata.UpstreamSource)
	}
}

func TestDiversionConsecutiveObservations(t *testing.T) {
	airportDB := GetAirportDB()
	engine := NewDiversionEngine(airportDB)

	origApt := airportDB.Lookup("ZBAA")
	destApt := airportDB.Lookup("ZSSS")
	if origApt == nil || destApt == nil {
		t.Fatalf("Failed to find ZBAA or ZSSS airport")
	}

	route := &models.RouteInfo{
		Summary:     "PEK -> SHA",
		Origin:      origApt,
		Destination: destApt,
	}

	// Flying Beijing -> Shanghai, but turns around heading north (Track 5°) towards Jinan Yaoqiang (ZSJN ~36.85, 117.21)
	deviatedAC := &models.RawAircraftSnapshot{
		Hex:          "dev001",
		Flight:       "TESTDEV",
		Squawk:       "1200",
		Lat:          36.6,
		Lon:          117.2,
		AltBaroFt:    7500,  // < 12,000 ft far from dest (> 600km)
		GroundSpeed:  420,
		Track:        5.0,   // heading north away from Shanghai (heading dev ~145° > 60°)
		VerticalRate: -1200, // descending towards Jinan
		OnGround:     false,
		LastContact:  1700000000,
	}

	// 1st observation: deviation count = 1, NOT sustained heading deviation
	res1 := engine.Analyze(deviatedAC, route)
	if res1.PredictedDiversion {
		t.Errorf("Single observation should not trigger diversion prediction (requires >= 2 consecutive)")
	}

	// 2nd consecutive observation: progressing timestamp, sustained heading deviation
	deviatedAC2 := *deviatedAC
	deviatedAC2.LastContact = 1700000030
	res2 := engine.Analyze(&deviatedAC2, route)
	if !res2.PredictedDiversion {
		t.Errorf("Expected 2nd consecutive deviated observation to trigger PredictedDiversion, got false")
	}
	if res2.CandidateAlternate == nil || res2.CandidateAlternate.ICAO != "ZSJN" {
		t.Errorf("Expected candidate alternate ZSJN, got %v", res2.CandidateAlternate)
	}

	// 3rd observation: Heading restored towards destination (Shanghai ~150°)
	alignedAC := &models.RawAircraftSnapshot{
		Hex:          "dev001",
		Flight:       "TESTDEV",
		Squawk:       "1200",
		Lat:          36.6,
		Lon:          117.2,
		AltBaroFt:    32000,
		GroundSpeed:  450,
		Track:        150.0,
		VerticalRate: 0,
		OnGround:     false,
		LastContact:  1700000060,
	}
	res3 := engine.Analyze(alignedAC, route)
	if res3.PredictedDiversion {
		t.Errorf("Restored heading should have cleared diversion prediction")
	}
}

func TestEstimatedPointDoesNotTriggerDiversion(t *testing.T) {
	airportDB := GetAirportDB()
	engine := NewDiversionEngine(airportDB)

	origApt := airportDB.Lookup("ZBAA")
	destApt := airportDB.Lookup("ZSSS")
	route := &models.RouteInfo{
		Summary:     "PEK -> SHA",
		Origin:      origApt,
		Destination: destApt,
	}

	estimatedAC := &models.RawAircraftSnapshot{
		Hex:          "est001",
		Flight:       "EST001",
		Squawk:       "7700", // even emergency squawk!
		Lat:          36.6,
		Lon:          117.2,
		AltBaroFt:    7500,
		GroundSpeed:  420,
		Track:        5.0,
		VerticalRate: -1200,
		OnGround:     false,
		Source:       "estimated", // Estimated dead-reckoning point
	}

	res := engine.Analyze(estimatedAC, route)
	if res.PredictedDiversion {
		t.Errorf("Estimated point must NEVER trigger diversion alert")
	}
	if res.ProbabilityLevel != "NONE" {
		t.Errorf("Expected probability NONE for estimated point, got %s", res.ProbabilityLevel)
	}

	staleAC := &models.RawAircraftSnapshot{
		Hex:    "stale001",
		Source: "stale",
	}
	resStale := engine.Analyze(staleAC, route)
	if resStale.PredictedDiversion || resStale.ProbabilityLevel != "NONE" {
		t.Errorf("Stale point must NEVER trigger diversion alert")
	}
}

func TestDynamicDeadReckoningMaxAge(t *testing.T) {
	cfg := &config.Config{
		OpenSkyMinIntervalSec:   100,
		DeadReckoningMaxAgeSec:  720,
		AircraftPurgeTimeoutSec: 900,
	}
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_dyn.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	p := NewPoller(cfg, db)

	// Simulate OpenSky scheduler with interval = 200s
	p.openSky.mu.Lock()
	now := time.Now().Unix()
	p.openSky.state = &models.UpstreamState{
		Provider:      "opensky",
		LastSuccessAt: now,
		NextAllowedAt: now + 200,
	}
	p.openSky.mu.Unlock()

	// Dynamic max age should be min(720, 200 * 1.2) = 240s
	maxAge := p.GetDynamicMaxAgeSec("opensky")
	if maxAge != 240.0 {
		t.Errorf("Expected dynamic max age 240.0s, got %f", maxAge)
	}

	// Verify fallback to cfg.DeadReckoningMaxAgeSec when interval not set
	maxAgeFallback := p.GetDynamicMaxAgeSec("unknown_source")
	if maxAgeFallback != 720.0 {
		t.Errorf("Expected fallback max age 720.0s, got %f", maxAgeFallback)
	}
}

func TestEstimatedCountInGlobalStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_stats.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	nowUnix := now.Unix()

	flights := []models.FlightData{
		// 1. Observed (elapsed 10s <= 30s)
		{
			Aircraft: models.AircraftInfo{Hex: "111111"},
			Telemetry: models.TelemetryInfo{
				Position:         models.PositionInfo{Latitude: 10, Longitude: 10},
				LastContactEpoch: nowUnix - 10,
			},
			Metadata: models.MetadataInfo{UpstreamSource: "opensky", LastObservedAt: nowUnix - 10},
		},
		// 2. Estimated (elapsed 120s: 30s < t <= 720s)
		{
			Aircraft: models.AircraftInfo{Hex: "222222"},
			Telemetry: models.TelemetryInfo{
				Position:         models.PositionInfo{Latitude: 20, Longitude: 20},
				LastContactEpoch: nowUnix - 120,
			},
			Metadata: models.MetadataInfo{UpstreamSource: "opensky", LastObservedAt: nowUnix - 120},
		},
		// 3. Stale (elapsed 800s > 720s)
		{
			Aircraft: models.AircraftInfo{Hex: "333333"},
			Telemetry: models.TelemetryInfo{
				Position:         models.PositionInfo{Latitude: 30, Longitude: 30},
				LastContactEpoch: nowUnix - 800,
			},
			Metadata: models.MetadataInfo{UpstreamSource: "opensky", LastObservedAt: nowUnix - 800},
		},
	}

	if err := db.SaveLiveFlights(flights); err != nil {
		t.Fatalf("SaveLiveFlights failed: %v", err)
	}

	stats, err := db.GetGlobalStats()
	if err != nil {
		t.Fatalf("GetGlobalStats failed: %v", err)
	}

	if stats["total_tracked"].(int) != 3 {
		t.Errorf("Expected total_tracked 3, got %v", stats["total_tracked"])
	}
	if stats["tracked_observed_count"].(int) != 1 {
		t.Errorf("Expected tracked_observed_count 1, got %v", stats["tracked_observed_count"])
	}
	if stats["tracked_estimated_count"].(int) != 1 {
		t.Errorf("Expected tracked_estimated_count 1, got %v", stats["tracked_estimated_count"])
	}
	if stats["tracked_stale_count"].(int) != 1 {
		t.Errorf("Expected tracked_stale_count 1, got %v", stats["tracked_stale_count"])
	}
}

func TestADSBMeshGlobalCooldownPersistence(t *testing.T) {
	cfg := &config.Config{
		ADSBMeshGlobalCooldownMin:  15,
		ADSBMeshRecoveryIntervalSec: 4.0,
	}
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_global_cd.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	mesh := NewADSBMeshScheduler(cfg, db, nil)
	now := time.Now()

	// Trigger global cooldown
	mesh.triggerGlobalCooldown(now)

	// Check DB persistence
	savedState, err := db.GetUpstreamState("adsb_mesh")
	if err != nil || savedState == nil {
		t.Fatalf("Expected saved upstream state in db: %v", err)
	}
	if savedState.Status != "cooldown" {
		t.Errorf("Expected status 'cooldown', got %s", savedState.Status)
	}
	expectedCooldownUnix := now.Add(15 * time.Minute).Unix()
	if savedState.CooldownUntil != expectedCooldownUnix {
		t.Errorf("Expected CooldownUntil %d, got %d", expectedCooldownUnix, savedState.CooldownUntil)
	}

	// Simulate restart by creating a new ADSBMeshScheduler instance
	meshRestarted := NewADSBMeshScheduler(cfg, db, nil)
	restartedStatus := meshRestarted.GetStatus()
	if restartedStatus["status"] != "cooldown" {
		t.Errorf("Expected restarted status to be 'cooldown', got %v", restartedStatus["status"])
	}
	if restartedStatus["active_hubs"].(int) != 0 {
		t.Errorf("Expected 0 active hubs in global cooldown, got %v", restartedStatus["active_hubs"])
	}
	if restartedStatus["cooldown_hubs"].(int) != restartedStatus["total_hubs"].(int) {
		t.Errorf("Expected cooldown_hubs == total_hubs in global cooldown, got %v vs %v",
			restartedStatus["cooldown_hubs"], restartedStatus["total_hubs"])
	}
	if restartedStatus["next_request_at"] != restartedStatus["cooldown_until"] {
		t.Errorf("Expected next_request_at == cooldown_until, got %v vs %v",
			restartedStatus["next_request_at"], restartedStatus["cooldown_until"])
	}
	if _, hasCredits := restartedStatus["remaining_credits"]; hasCredits {
		t.Errorf("Expected NO remaining_credits in adsb_mesh status, got %v", restartedStatus["remaining_credits"])
	}
	if meshRestarted.globalCooldownFailures != 1 {
		t.Errorf("Expected restored globalCooldownFailures 1, got %d", meshRestarted.globalCooldownFailures)
	}
}

func TestADSBMeshStatusWhenAllHubsCooldown(t *testing.T) {
	cfg := &config.Config{}
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_all_cd.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	mesh := NewADSBMeshScheduler(cfg, db, nil)
	now := time.Now()

	// Put all hubs into cooldown
	mesh.mu.Lock()
	for _, hs := range mesh.hubStates {
		hs.CooldownUntil = now.Add(10 * time.Minute)
	}
	mesh.mu.Unlock()

	status := mesh.GetStatus()
	// Active hubs is 0 -> status MUST be "cooldown", NOT "ok"
	if status["active_hubs"].(int) != 0 {
		t.Errorf("Expected 0 active hubs, got %v", status["active_hubs"])
	}
	if status["status"] != "cooldown" {
		t.Errorf("Expected status 'cooldown' when all hubs are cooling down, got %v", status["status"])
	}
}

func TestBuildWeightedHubSequence(t *testing.T) {
	nowUTC := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	seq := BuildWeightedHubSequence(defaultGlobalHubs, nowUTC)

	if len(seq) == 0 {
		t.Fatalf("Expected non-empty sequence")
	}

	// All default global hubs must appear at least once (no starvation)
	seen := make(map[string]int)
	for _, h := range seq {
		seen[h.Name]++
	}

	for _, h := range defaultGlobalHubs {
		if seen[h.Name] == 0 {
			t.Errorf("Hub %s was starved (never appeared in sequence)", h.Name)
		}
	}

	// Verify that total hubs is 69
	if len(defaultGlobalHubs) != 69 {
		t.Errorf("Expected 69 global hubs, got %d", len(defaultGlobalHubs))
	}
}

func TestOpenSkyGlobalBudgetFormula(t *testing.T) {
	// 4000 daily limit, 0.65 global share = 2600 credits.
	// 4 credits/call -> 650 calls/day.
	// 86400 / 650 = 132.92s ~ 133s.
	interval := CalculateOpenSkyGlobalInterval(2600, 2600, 4, 100)
	if interval != 133*time.Second {
		t.Errorf("Expected 133s uniform interval for 2600 budget, got %v", interval)
	}

	// 20% threshold is 520 credits. If remaining is 100 credits, interval slows down smoothly towards 900s.
	slowInterval := CalculateOpenSkyGlobalInterval(100, 2600, 4, 100)
	if slowInterval < 500*time.Second || slowInterval > 900*time.Second {
		t.Errorf("Expected interval to slow down significantly when remaining is 100 (<20%%), got %v", slowInterval)
	}
}

func TestADSBLolCalibrationQueuePriority(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_cal_queue.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{
		ADSBLolDailyLimit:             450,
		ADSBLolMinIntervalSec:         30,
		ADSBLolCalibrationCooldownSec: 600,
	}

	sched := NewADSBMeshScheduler(cfg, db, nil)

	// Enqueue tasks with different priorities:
	// Priority 4: Stale flight
	sched.EnqueueCalibration("aaaaaa", 4)
	// Priority 2: Unverified anomaly
	sched.EnqueueCalibration("bbbbbb", 2)
	// Priority 3: Approaching flight
	sched.EnqueueCalibration("cccccc", 3)

	if len(sched.calibrationQueue) != 3 {
		t.Fatalf("Expected 3 items in calibration queue, got %d", len(sched.calibrationQueue))
	}

	// Deduplication test: re-enqueuing "aaaaaa" within cooldown should be ignored
	sched.EnqueueCalibration("aaaaaa", 1)
	if len(sched.calibrationQueue) != 3 {
		t.Errorf("Expected queue length to remain 3 after duplicate hex, got %d", len(sched.calibrationQueue))
	}
}

func TestHoldingPatternSuppressesDiversion(t *testing.T) {
	airportDB := GetAirportDB()
	engine := NewDiversionEngine(airportDB)

	destApt := airportDB.Lookup("ZBAA")
	if destApt == nil {
		t.Fatalf("Failed to find ZBAA airport")
	}

	route := &models.RouteInfo{
		Summary: "SHA -> PEK",
		Origin: &models.AirportInfo{
			ICAO: "ZSSS",
			Name: "Shanghai Hongqiao",
			Location: models.AirportLocation{
				Latitude: 31.19, Longitude: 121.33,
			},
		},
		Destination: destApt,
	}

	// Simulate aircraft performing 4 consecutive 90-degree turns within a 5 km localized area
	hex := "hold01"
	centerLat, centerLon := 36.5, 117.5

	var lastAnalysis models.DiversionAnalysis
	for i := 0; i < 5; i++ {
		heading := float64(i * 90)
		dLat := 0.02 * math.Sin(float64(i)*math.Pi/2.0)
		dLon := 0.02 * math.Cos(float64(i)*math.Pi/2.0)

		snap := &models.RawAircraftSnapshot{
			Hex:          hex,
			Flight:       "CSN333",
			Squawk:       "1200",
			Lat:          centerLat + dLat,
			Lon:          centerLon + dLon,
			AltBaroFt:    15000,
			GroundSpeed:  220,
			Track:        heading,
			VerticalRate: 0,
			OnGround:     false,
			LastContact:  time.Now().Unix() + int64(i*30),
		}
		lastAnalysis = engine.Analyze(snap, route)
	}

	if !lastAnalysis.IsHolding {
		t.Errorf("Expected flight in 360-degree circle to be marked as holding, got is_holding=%v", lastAnalysis.IsHolding)
	}
	if lastAnalysis.PredictedDiversion {
		t.Errorf("Expected holding pattern to suppress diversion prediction, got predicted_diversion=true")
	}
}

func TestCorridorDeviationCheck(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_corridor.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// Save learned corridor with 10 samples for SHA -> PEK
	segments := []database.CorridorSegment{
		{ProgressPercent: 0.25, CenterLat: 33.0, CenterLon: 120.0, WidthKm: 30.0},
		{ProgressPercent: 0.50, CenterLat: 35.5, CenterLon: 118.5, WidthKm: 35.0},
		{ProgressPercent: 0.75, CenterLat: 38.0, CenterLon: 117.5, WidthKm: 30.0},
	}
	if err := db.SaveRouteCorridor("ZSSS", "ZBAA", 10, segments); err != nil {
		t.Fatalf("SaveRouteCorridor failed: %v", err)
	}

	corridor, found, err := db.GetRouteCorridor("ZSSS", "ZBAA")
	if err != nil || !found {
		t.Fatalf("Failed to retrieve learned route corridor: %v", err)
	}
	if corridor.SampleCount != 10 || len(corridor.Segments) != 3 {
		t.Errorf("Corridor data mismatch: count=%d, segments=%d", corridor.SampleCount, len(corridor.Segments))
	}
}

func TestASA286RouteValidationAndDiversionSuppression(t *testing.T) {
	airportDB := GetAirportDB()
	engine := NewDiversionEngine(airportDB)

	// Route: LAX -> EWR
	laxApt := &models.AirportInfo{
		ICAO: "KLAX",
		IATA: "LAX",
		Name: "Los Angeles International Airport",
		Location: models.AirportLocation{
			Latitude:  33.9425,
			Longitude: -118.4081,
		},
	}
	ewrApt := &models.AirportInfo{
		ICAO: "KEWR",
		IATA: "EWR",
		Name: "Newark Liberty International Airport",
		Location: models.AirportLocation{
			Latitude:  40.6925,
			Longitude: -74.1687,
		},
	}
	route := &models.RouteInfo{
		Summary:     "LAX -> EWR",
		Origin:      laxApt,
		Destination: ewrApt,
	}

	// Case 1: ASA286 in Central California flying South (Track 167°), deviation 96° from destination bearing (~71°)
	// Level flight at FL360, no emergency. Route schedule is outdated/reused.
	asa286 := &models.RawAircraftSnapshot{
		Hex:          "a1b2c3",
		Flight:       "ASA286",
		Squawk:       "1200",
		Lat:          36.2,
		Lon:          -119.8, // Central California
		AltBaroFt:    36000,
		GroundSpeed:  460,
		Track:        167.0, // Heading South
		VerticalRate: 0,
		OnGround:     false,
		LastContact:  time.Now().Unix(),
	}

	analysis := engine.Analyze(asa286, route)
	if analysis.PredictedDiversion {
		t.Errorf("ASA286 with outdated route should NOT predict diversion, got predicted_diversion=true")
	}
	if analysis.StatusTier != "none" {
		t.Errorf("Expected status tier 'none' for rejected outdated route, got %s", analysis.StatusTier)
	}

	// Case 2: Flight on reverse return leg (e.g. RPA4686 flying return leg)
	// Aircraft in cruise FL280 flying towards listed Origin (LAX)
	rpa4686 := &models.RawAircraftSnapshot{
		Hex:          "c4d5e6",
		Flight:       "RPA4686",
		Squawk:       "1200",
		Lat:          35.5,
		Lon:          -115.0,
		AltBaroFt:    28000,
		GroundSpeed:  440,
		Track:        245.0, // Heading directly towards LAX
		VerticalRate: 0,
		OnGround:     false,
		LastContact:  time.Now().Unix(),
	}

	returnRoute := &models.RouteInfo{
		Summary:     "LAX -> EWR",
		Origin:      laxApt,
		Destination: ewrApt,
	}
	analysisReturn := engine.Analyze(rpa4686, returnRoute)
	if analysisReturn.PredictedDiversion {
		t.Errorf("Inverted return leg flight should NOT trigger diversion, got predicted_diversion=true")
	}
}

func TestCalculateOpenSkyAreaCost(t *testing.T) {
	// Area <= 25 deg^2: 1 credit (e.g., 4 x 5 = 20)
	if cost := CalculateOpenSkyAreaCost(30.0, 110.0, 34.0, 115.0); cost != 1 {
		t.Errorf("Expected cost 1 for area 20 deg^2, got %d", cost)
	}

	// Area 25 - 100 deg^2: 2 credits (e.g., 6 x 8 = 48)
	if cost := CalculateOpenSkyAreaCost(30.0, 110.0, 36.0, 118.0); cost != 2 {
		t.Errorf("Expected cost 2 for area 48 deg^2, got %d", cost)
	}

	// Area 100 - 400 deg^2: 3 credits (e.g., 10 x 20 = 200)
	if cost := CalculateOpenSkyAreaCost(30.0, 100.0, 40.0, 120.0); cost != 3 {
		t.Errorf("Expected cost 3 for area 200 deg^2, got %d", cost)
	}

	// Area > 400 deg^2 or global: 4 credits (e.g., 25 x 25 = 625)
	if cost := CalculateOpenSkyAreaCost(10.0, 90.0, 35.0, 115.0); cost != 4 {
		t.Errorf("Expected cost 4 for area 625 deg^2, got %d", cost)
	}
}

