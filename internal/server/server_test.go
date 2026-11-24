package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"contrail/internal/config"
	"contrail/internal/database"
	"contrail/internal/models"
	"contrail/internal/poller"
)

const (
	testAdminEmail    = "admin@example.test"
	testAdminPassword = "test-admin-password"
)

func init() {
	os.Setenv("ADMIN_EMAIL", testAdminEmail)
	os.Setenv("ADMIN_PASSWORD", testAdminPassword)
}

func setupTestServer(t *testing.T) (*Server, *database.DB, func()) {
	t.Setenv("ENV", "test")
	tmpDir, err := os.MkdirTemp("", "contrail-server-test-*")
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Port:           ":8080",
		APIBaseURL:     "https://airline.api.veyino.org",
		DBPath:         dbPath,
		RateLimitRPM:   60,
		RateLimitRPH:   1000,
		RateLimitDaily: 10000,
		JWTSecret:      "test-jwt-secret-key-123",
		CapSiteKey:     "b12b141f33",
		CapSecretKey:   "test-cap-secret",
		CapAPIEndpoint: "/cap/b12b141f33/",
		CapVerifyURL:   "http://127.0.0.1:13000/b12b141f33/siteverify",
	}

	p := poller.NewPoller(cfg, db)
	srv := NewServer(cfg, db, p)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return srv, db, cleanup
}

func strPtr(s string) *string {
	return &s
}

func TestServerEndpoints(t *testing.T) {
	srv, db, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert sample flight
	emergencyFlight := models.FlightData{
		Aircraft: models.AircraftInfo{
			Hex: "39de4a",
		},
		Flight: models.FlightIdentifier{
			CallsignICAO: "AFR123",
			CallsignIATA: strPtr("AF123"),
		},
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  48.8566,
				Longitude: 2.3522,
			},
			Altitude: models.AltitudeInfo{
				BarometricFt: 8000,
			},
			Speed: models.SpeedInfo{
				GroundSpeedKts: 250,
			},
			OnGround: false,
		},
		Emergency: models.EmergencyInfo{
			Squawk:      "7700",
			IsEmergency: true,
			Category:    models.EmergencyGeneral,
			Description: "Mayday",
		},
		DiversionAnalysis: models.DiversionAnalysis{
			PredictedDiversion: true,
			ProbabilityLevel:   "CRITICAL",
		},
		Metadata: models.MetadataInfo{
			UpstreamSource: "adsb_fi",
			LastUpdated:    time.Now(),
		},
	}
	db.SaveLiveFlights([]models.FlightData{emergencyFlight})

	// Test 1: GET /api/v1/health (Public)
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK from /health, got %d", w.Code)
	}

	var healthRes models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &healthRes); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}
	if healthRes.Status != "success" {
		t.Errorf("Expected status success, got %s", healthRes.Status)
	}
	if healthRes.Attribution == nil || healthRes.Attribution.Citation == "" {
		t.Errorf("Expected attribution notice in response, got nil or empty")
	}

	// Test 2: GET /api/v1/flights WITHOUT API key -> Must be rejected with 401
	reqNoKey := httptest.NewRequest("GET", "/api/v1/flights?emergency=true", nil)
	wNoKey := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wNoKey, reqNoKey)

	if wNoKey.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for unauthenticated call, got %d", wNoKey.Code)
	}

	// Provision test user and key
	user, err := db.CreateUser("Test Researcher", "researcher@example.com", "Password123!", "Aircraft research", true)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	rawKey, apiKey, err := db.GenerateAPIKey(user.ID, "Testing Key")
	if err != nil {
		t.Fatalf("Failed to generate test API key: %v", err)
	}

	// Test 3: GET /api/v1/flights WITH API key -> Must succeed with 200 OK
	reqWithKey := httptest.NewRequest("GET", "/api/v1/flights?emergency=true", nil)
	reqWithKey.Header.Set("X-API-Key", rawKey)
	wWithKey := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wWithKey, reqWithKey)

	if wWithKey.Code != http.StatusOK {
		t.Errorf("Expected 200 from /flights with key, got %d", wWithKey.Code)
	}
	var flightsRes models.APIResponse
	json.Unmarshal(wWithKey.Body.Bytes(), &flightsRes)
	if flightsRes.FilteredCount != 1 {
		t.Errorf("Expected 1 emergency flight, got %d", flightsRes.FilteredCount)
	}
	if flightsRes.Attribution == nil {
		t.Errorf("Expected attribution metadata in flight list response")
	}

	// Test 4: POST /api/v1/query WITH API key
	queryPayload := models.QueryRequest{
		Filters: models.FlightQueryFilter{
			CallsignPrefix: "AFR",
		},
		Limit: 10,
	}
	bodyBytes, _ := json.Marshal(queryPayload)
	req = httptest.NewRequest("POST", "/api/v1/query", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", rawKey)
	w = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 from /query, got %d", w.Code)
	}
	var queryRes models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &queryRes)
	if queryRes.FilteredCount != 1 {
		t.Errorf("Expected 1 matching flight for query, got %d", queryRes.FilteredCount)
	}

	// Test 5: Key Revocation -> Revoking key must cause subsequent requests to fail with 401
	err = db.RevokeAPIKey(apiKey.ID, user.ID, false)
	if err != nil {
		t.Fatalf("Failed to revoke key: %v", err)
	}
	reqRevoked := httptest.NewRequest("GET", "/api/v1/flights", nil)
	reqRevoked.Header.Set("X-API-Key", rawKey)
	wRevoked := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wRevoked, reqRevoked)
	if wRevoked.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for revoked key, got %d", wRevoked.Code)
	}

	// Test 6: Self-hosted CAPTCHA generation
	reqCaptcha := httptest.NewRequest("GET", "/api/v1/auth/captcha", nil)
	wCaptcha := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wCaptcha, reqCaptcha)
	if wCaptcha.Code != http.StatusOK {
		t.Errorf("Expected 200 for captcha generation, got %d", wCaptcha.Code)
	}

	// Test 7: GET / (Welcome page HTML)
	reqHome := httptest.NewRequest("GET", "/", nil)
	reqHome.Header.Set("Accept", "text/html")
	wHome := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wHome, reqHome)
	if wHome.Code != http.StatusOK {
		t.Errorf("Expected 200 for welcome page, got %d", wHome.Code)
	}

	// Test 8: GET /data-notice (HTML and raw markdown)
	reqNoticeHTML := httptest.NewRequest("GET", "/data-notice", nil)
	reqNoticeHTML.Header.Set("Accept", "text/html")
	wNoticeHTML := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wNoticeHTML, reqNoticeHTML)
	if wNoticeHTML.Code != http.StatusOK {
		t.Errorf("Expected 200 for /data-notice HTML, got %d", wNoticeHTML.Code)
	}
	if !strings.Contains(wNoticeHTML.Header().Get("Content-Type"), "text/html") {
		t.Errorf("Expected text/html content-type, got %s", wNoticeHTML.Header().Get("Content-Type"))
	}
	if !strings.Contains(wNoticeHTML.Body.String(), "Data Notice") {
		t.Errorf("Expected Data Notice in body")
	}

	reqNoticeRaw := httptest.NewRequest("GET", "/data-notice?format=raw", nil)
	wNoticeRaw := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wNoticeRaw, reqNoticeRaw)
	if wNoticeRaw.Code != http.StatusOK {
		t.Errorf("Expected 200 for /data-notice raw, got %d", wNoticeRaw.Code)
	}
	if !strings.Contains(wNoticeRaw.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("Expected text/plain content-type, got %s", wNoticeRaw.Header().Get("Content-Type"))
	}

	// Test 9: Key Hard Deletion Verification
	// Ensure revoked key is deleted from database
	userKeys, err := db.GetUserAPIKeys(user.ID)
	if err != nil {
		t.Fatalf("Failed to fetch user keys: %v", err)
	}
	for _, k := range userKeys {
		if k.ID == apiKey.ID {
			t.Errorf("Expected key %d to be hard deleted from user keys, but still found", apiKey.ID)
		}
	}

	// Test 10: Admin Upstream Logs Endpoint
	// Add test log into db
	err = db.LogUpstreamFetch("opensky", "global", "rate_limited", 429, 0, 50, "Rate limited (429)")
	if err != nil {
		t.Fatalf("Failed to log upstream fetch: %v", err)
	}
	err = db.LogUpstreamFetch("adsb_mesh", "ZBAA", "success", 200, 1500, 1200, "Aggregated nodes")
	if err != nil {
		t.Fatalf("Failed to log upstream fetch: %v", err)
	}

	// Non-admin user should get 403 Forbidden
	normalToken, _ := srv.auth.GenerateJWT(user)
	reqLogsNormal := httptest.NewRequest("GET", "/api/v1/admin/upstream-logs", nil)
	reqLogsNormal.Header.Set("Authorization", "Bearer "+normalToken)
	wLogsNormal := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wLogsNormal, reqLogsNormal)
	if wLogsNormal.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for normal user accessing upstream logs, got %d", wLogsNormal.Code)
	}

	// Admin user should get 200 OK
	adminUser, err := db.GetUserByEmail(testAdminEmail)
	if err != nil {
		t.Fatalf("Failed to get admin user: %v", err)
	}
	adminToken, _ := srv.auth.GenerateJWT(adminUser)
	reqLogsAdmin := httptest.NewRequest("GET", "/api/v1/admin/upstream-logs?limit=10", nil)
	reqLogsAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	wLogsAdmin := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wLogsAdmin, reqLogsAdmin)
	if wLogsAdmin.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for admin accessing upstream logs, got %d", wLogsAdmin.Code)
	}
	var upstreamRes struct {
		Status string                      `json:"status"`
		Data   []*database.UpstreamFetchLog `json:"data"`
	}
	if err := json.Unmarshal(wLogsAdmin.Body.Bytes(), &upstreamRes); err != nil {
		t.Fatalf("Failed to unmarshal upstream logs: %v", err)
	}
	if len(upstreamRes.Data) < 2 {
		t.Errorf("Expected at least 2 upstream logs, got %d", len(upstreamRes.Data))
	}

	// Test 11: Rate Limit Headers in API Responses
	testRawKey, _, err := db.GenerateAPIKey(user.ID, "Rate Limit Test Key")
	if err != nil {
		t.Fatalf("Failed to create rate limit test key: %v", err)
	}

	reqFlightWithKey := httptest.NewRequest("GET", "/api/v1/flights?limit=1", nil)
	reqFlightWithKey.Header.Set("X-API-Key", testRawKey)
	wFlightWithKey := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wFlightWithKey, reqFlightWithKey)
	if wFlightWithKey.Code != http.StatusOK {
		t.Errorf("Expected 200 OK with valid API key, got %d", wFlightWithKey.Code)
	}
	if wFlightWithKey.Header().Get("X-RateLimit-Limit") == "" {
		t.Errorf("Expected X-RateLimit-Limit header in response")
	}
	if wFlightWithKey.Header().Get("X-RateLimit-Remaining") == "" {
		t.Errorf("Expected X-RateLimit-Remaining header in response")
	}
	if wFlightWithKey.Header().Get("X-RateLimit-Reset") == "" {
		t.Errorf("Expected X-RateLimit-Reset header in response")
	}
	if wFlightWithKey.Header().Get("X-RateLimit-Limit-Daily") == "" {
		t.Errorf("Expected X-RateLimit-Limit-Daily header in response")
	}
	if wFlightWithKey.Header().Get("X-RateLimit-Remaining-Daily") == "" {
		t.Errorf("Expected X-RateLimit-Remaining-Daily header in response")
	}
	if wFlightWithKey.Header().Get("X-RateLimit-Daily-Reset") == "" {
		t.Errorf("Expected X-RateLimit-Daily-Reset header in response")
	}
}

func TestRegistrationChannelStatus(t *testing.T) {
	srv, db, cleanup := setupTestServer(t)
	defer cleanup()

	// Create admin user
	adminUser, err := db.CreateUser("Admin", "admin@contrail.local", "AdminPass123!", "Platform Admin", true)
	if err != nil {
		t.Fatalf("Failed to create admin: %v", err)
	}
	_, _ = db.Exec("UPDATE users SET role = 'admin' WHERE id = ?", adminUser.ID)
	adminUser.Role = "admin"
	adminToken, _ := srv.auth.GenerateJWT(adminUser)

	// Step 1: Query initial registration status (should be true)
	reqStatus := httptest.NewRequest("GET", "/api/v1/admin/registration-status", nil)
	reqStatus.Header.Set("Authorization", "Bearer "+adminToken)
	wStatus := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wStatus, reqStatus)

	if wStatus.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from registration status, got %d", wStatus.Code)
	}
	var res models.APIResponse
	json.Unmarshal(wStatus.Body.Bytes(), &res)
	statusData := res.Data.(map[string]interface{})
	if statusData["registration_enabled"] != true {
		t.Errorf("Expected registration_enabled=true initially, got %v", statusData["registration_enabled"])
	}

	// Step 2: Disable registration channel
	disablePayload := []byte(`{"registration_enabled":false}`)
	reqDisable := httptest.NewRequest("POST", "/api/v1/admin/registration-status", bytes.NewReader(disablePayload))
	reqDisable.Header.Set("Authorization", "Bearer "+adminToken)
	wDisable := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wDisable, reqDisable)

	if wDisable.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK when disabling registration, got %d", wDisable.Code)
	}

	// Step 3: When disabled, requesting code sends notice email without storing verification code
	sendCodePayload := []byte(`{"email":"applicant@example.com"}`)
	reqSendCode := httptest.NewRequest("POST", "/api/v1/auth/send-code", bytes.NewReader(sendCodePayload))
	wSendCode := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wSendCode, reqSendCode)

	if wSendCode.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from send-code when disabled, got %d", wSendCode.Code)
	}

	// Verify NO code was generated or stored in email_verifications for this applicant
	var codeCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM email_verifications WHERE email = 'applicant@example.com'").Scan(&codeCount)
	if codeCount != 0 {
		t.Errorf("Expected 0 verification codes stored when registration is disabled, got %d", codeCount)
	}

	// Step 4: Direct register attempt must be blocked with 403 Forbidden
	regPayload := []byte(`{"name":"New User","email":"applicant@example.com","password":"Password123!","purpose":"Academic","agree_terms":true,"verification_code":"123456"}`)
	reqReg := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(regPayload))
	wReg := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wReg, reqReg)

	if wReg.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden when registering with disabled channel, got %d", wReg.Code)
	}

	// Step 5: Re-enable channel
	enablePayload := []byte(`{"registration_enabled":true}`)
	reqEnable := httptest.NewRequest("POST", "/api/v1/admin/registration-status", bytes.NewReader(enablePayload))
	reqEnable.Header.Set("Authorization", "Bearer "+adminToken)
	wEnable := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wEnable, reqEnable)

	if wEnable.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK when re-enabling channel, got %d", wEnable.Code)
	}

	// Send code should now store code
	reqSendCode2 := httptest.NewRequest("POST", "/api/v1/auth/send-code", bytes.NewReader(sendCodePayload))
	wSendCode2 := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wSendCode2, reqSendCode2)

	_ = db.QueryRow("SELECT COUNT(*) FROM email_verifications WHERE email = 'applicant@example.com'").Scan(&codeCount)
	if codeCount != 1 {
		t.Errorf("Expected 1 verification code stored after channel re-enabled, got %d", codeCount)
	}
}

func TestFieldRestrictionsPermissionIsolation(t *testing.T) {
	srv, db, cleanup := setupTestServer(t)
	defer cleanup()

	// Seed test flight
	flight := models.FlightData{
		Aircraft: models.AircraftInfo{
			Hex:          "394a10",
			Registration: "F-GZTA",
			Type:         "B772",
		},
		Flight: models.FlightIdentifier{
			CallsignICAO: "AFR123",
		},
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  48.85,
				Longitude: 2.35,
			},
			Altitude: models.AltitudeInfo{
				BarometricFt: 30000,
			},
			Speed: models.SpeedInfo{
				GroundSpeedKts: 450,
			},
			OnGround: false,
		},
		Emergency: models.EmergencyInfo{
			IsEmergency: true,
			Squawk:      "7700",
			Description: "General Emergency",
		},
		DiversionAnalysis: models.DiversionAnalysis{
			PredictedDiversion: true,
		},
		Route: models.RouteInfo{
			Summary: "CDG -> JFK",
			Origin: &models.AirportInfo{
				ICAO: "LFPG",
				Name: "Paris Charles de Gaulle",
			},
			Destination: &models.AirportInfo{
				ICAO: "KJFK",
				Name: "John F Kennedy International",
			},
		},
	}
	db.SaveLiveFlights([]models.FlightData{flight})

	// Create normal user and key
	normalUser, _ := db.CreateUser("Normal User", "user@example.com", "UserPass123!", "Research", true)
	normalRawKey, _, _ := db.GenerateAPIKey(normalUser.ID, "Normal Key")

	// Create admin user and key
	adminUser, _ := db.CreateUser("Admin User", "admin2@example.com", "AdminPass123!", "Admin", true)
	_, _ = db.Exec("UPDATE users SET role = 'admin' WHERE id = ?", adminUser.ID)
	adminUser.Role = "admin"
	adminRawKey, _, _ := db.GenerateAPIKey(adminUser.ID, "Admin Key")

	// Set allowed fields to ONLY ["aircraft"]
	if err := db.SetAllowedFieldsForNormalUsers([]string{"aircraft"}); err != nil {
		t.Fatalf("Failed to set allowed fields: %v", err)
	}

	// 1. GET /api/v1/flights: returns only aircraft
	reqFlights := httptest.NewRequest("GET", "/api/v1/flights", nil)
	reqFlights.Header.Set("X-API-Key", normalRawKey)
	wFlights := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wFlights, reqFlights)

	if wFlights.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /flights, got %d", wFlights.Code)
	}
	var fRes models.APIResponse
	json.Unmarshal(wFlights.Body.Bytes(), &fRes)
	flightsList, ok := fRes.Data.([]interface{})
	if !ok || len(flightsList) == 0 {
		t.Fatalf("Expected flights array, got %v", fRes.Data)
	}
	fMap := flightsList[0].(map[string]interface{})
	if fMap["aircraft"] == nil {
		t.Errorf("Expected 'aircraft' field to be present")
	}
	if fMap["telemetry"] != nil || fMap["route"] != nil || fMap["flight"] != nil || fMap["emergency"] != nil {
		t.Errorf("Disallowed fields were leaked in /flights: %v", fMap)
	}

	// 2. GET /api/v1/history: must return 403 Forbidden because telemetry is disallowed
	reqHist := httptest.NewRequest("GET", "/api/v1/history?hex=394a10", nil)
	reqHist.Header.Set("X-API-Key", normalRawKey)
	wHist := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wHist, reqHist)

	if wHist.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden from /history without telemetry permission, got %d", wHist.Code)
	}

	// 3. GET /api/v1/emergencies: must return 403 Forbidden because emergency is disallowed
	reqEmerg := httptest.NewRequest("GET", "/api/v1/emergencies", nil)
	reqEmerg.Header.Set("X-API-Key", normalRawKey)
	wEmerg := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wEmerg, reqEmerg)

	if wEmerg.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden from /emergencies without emergency permission, got %d", wEmerg.Code)
	}

	// 4. GET /api/v1/diversions: must return 403 Forbidden because diversion_analysis is disallowed
	reqDiv := httptest.NewRequest("GET", "/api/v1/diversions", nil)
	reqDiv.Header.Set("X-API-Key", normalRawKey)
	wDiv := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wDiv, reqDiv)

	if wDiv.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden from /diversions without diversion_analysis permission, got %d", wDiv.Code)
	}

	// 5. GET /api/v1/flights?emergency=true: must return 403 Forbidden
	reqFilterEmerg := httptest.NewRequest("GET", "/api/v1/flights?emergency=true", nil)
	reqFilterEmerg.Header.Set("X-API-Key", normalRawKey)
	wFilterEmerg := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wFilterEmerg, reqFilterEmerg)

	if wFilterEmerg.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden when filtering by emergency without permission, got %d", wFilterEmerg.Code)
	}

	// 6. GET /api/v1/airports/LFPG: active_inbound and active_outbound must be omitted
	reqApt := httptest.NewRequest("GET", "/api/v1/airports/LFPG", nil)
	reqApt.Header.Set("X-API-Key", normalRawKey)
	wApt := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wApt, reqApt)

	if wApt.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /airports/LFPG, got %d", wApt.Code)
	}
	var aptRes models.APIResponse
	json.Unmarshal(wApt.Body.Bytes(), &aptRes)
	aptMap := aptRes.Data.(map[string]interface{})
	if aptMap["active_inbound"] != nil || aptMap["active_outbound"] != nil {
		t.Errorf("Expected active_inbound and active_outbound to be omitted when route is disallowed, got %v", aptMap)
	}

	// 7. GET /api/v1/stats: top_origins, top_destinations, airborne_count, emergency_count must be stripped
	reqStats := httptest.NewRequest("GET", "/api/v1/stats", nil)
	reqStats.Header.Set("X-API-Key", normalRawKey)
	wStats := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wStats, reqStats)

	if wStats.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /stats, got %d", wStats.Code)
	}
	var statsRes models.APIResponse
	json.Unmarshal(wStats.Body.Bytes(), &statsRes)
	sMap := statsRes.Data.(map[string]interface{})
	if sMap["top_origins"] != nil || sMap["top_destinations"] != nil {
		t.Errorf("top_origins or top_destinations leaked without route permission: %v", sMap)
	}
	if sMap["airborne_count"] != nil || sMap["on_ground_count"] != nil {
		t.Errorf("airborne_count leaked without telemetry permission: %v", sMap)
	}
	if sMap["emergency_count"] != nil || sMap["emergencies"] != nil {
		t.Errorf("emergency_count leaked without emergency permission: %v", sMap)
	}
	if sMap["diverting"] != nil || sMap["predicted_diversion_count"] != nil {
		t.Errorf("diverting leaked without diversion_analysis permission: %v", sMap)
	}

	// 8. Admin Key Verification: Admin bypasses all field restrictions
	reqAdminEmerg := httptest.NewRequest("GET", "/api/v1/emergencies", nil)
	reqAdminEmerg.Header.Set("X-API-Key", adminRawKey)
	wAdminEmerg := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wAdminEmerg, reqAdminEmerg)

	if wAdminEmerg.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for admin on /emergencies, got %d", wAdminEmerg.Code)
	}
}
