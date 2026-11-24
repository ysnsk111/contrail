package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"contrail/internal/models"
)

func TestComplianceAndRegistrationFlow(t *testing.T) {
	srv, db, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Test Captcha Generation
	reqCap := httptest.NewRequest("GET", "/api/v1/auth/captcha", nil)
	wCap := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wCap, reqCap)

	if wCap.Code != http.StatusOK {
		t.Fatalf("Expected 200 from /api/v1/auth/captcha, got %d", wCap.Code)
	}

	var capResp models.APIResponse
	if err := json.Unmarshal(wCap.Body.Bytes(), &capResp); err != nil {
		t.Fatalf("Failed to decode captcha response: %v", err)
	}

	capData, _ := json.Marshal(capResp.Data)
	var challenge struct {
		CaptchaID string `json:"captcha_id"`
		ImageSVG  string `json:"image_svg"`
	}
	json.Unmarshal(capData, &challenge)

	if challenge.CaptchaID == "" || challenge.ImageSVG == "" {
		t.Fatalf("Expected valid captcha challenge, got %+v", challenge)
	}

	// 2. Test Verification Code Flow (Pre-store valid code in DB to simulate email send)
	testEmail := "pilot@example.com"
	testCode := "123456"
	err := db.SaveVerificationCode(testEmail, testCode, "register", 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to save verification code: %v", err)
	}

	// 3. Test Registration without agreeing to terms -> Must Fail (400)
	regPayloadNoTerms := models.RegisterRequest{
		Name:             "Commercial Operator Inc",
		Email:            testEmail,
		Password:         "SecurePassword123!",
		Purpose:          "Fleet logistics",
		VerificationCode: testCode,
		AgreeTerms:       false,
	}
	bodyBytes, _ := json.Marshal(regPayloadNoTerms)
	reqReg := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(bodyBytes))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wReg, reqReg)

	if wReg.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request when terms not agreed, got %d", wReg.Code)
	}

	// 4. Test Registration with terms agreed -> Must Succeed (201 Created)
	regPayloadSuccess := models.RegisterRequest{
		Name:             "Open Academic Research Team",
		Email:            testEmail,
		Password:         "SecurePassword123!",
		Purpose:          "Atmospheric science ADS-B telemetry analysis",
		VerificationCode: testCode,
		AgreeTerms:       true,
	}
	bodyBytes, _ = json.Marshal(regPayloadSuccess)
	reqReg = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(bodyBytes))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wReg, reqReg)

	if wReg.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for valid registration, got %d: %s", wReg.Code, wReg.Body.String())
	}

	// 5. Test Login
	loginPayload := models.LoginRequest{
		Email:    testEmail,
		Password: "SecurePassword123!",
	}
	bodyBytes, _ = json.Marshal(loginPayload)
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(bodyBytes))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for login, got %d", wLogin.Code)
	}

	var loginResp models.APIResponse
	json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	loginDataBytes, _ := json.Marshal(loginResp.Data)
	var loginData struct {
		Token string       `json:"token"`
		User  *models.User `json:"user"`
	}
	json.Unmarshal(loginDataBytes, &loginData)
	userToken := loginData.Token

	if userToken == "" || loginData.User == nil {
		t.Fatalf("Expected valid JWT token and user info, got empty")
	}

	// 6. Test Key Generation via Portal
	keyReqPayload := models.CreateAPIKeyRequest{
		KeyName:    "Production Telemetry Key",
		AgreeTerms: true,
	}
	bodyBytes, _ = json.Marshal(keyReqPayload)
	reqGenKey := httptest.NewRequest("POST", "/api/v1/portal/keys", bytes.NewReader(bodyBytes))
	reqGenKey.Header.Set("Content-Type", "application/json")
	reqGenKey.Header.Set("Authorization", "Bearer "+userToken)
	wGenKey := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wGenKey, reqGenKey)

	if wGenKey.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for key generation, got %d: %s", wGenKey.Code, wGenKey.Body.String())
	}

	var keyGenResp models.APIResponse
	json.Unmarshal(wGenKey.Body.Bytes(), &keyGenResp)
	kgBytes, _ := json.Marshal(keyGenResp.Data)
	var kgData struct {
		RawKey string         `json:"raw_key"`
		Key    *models.APIKey `json:"key"`
	}
	json.Unmarshal(kgBytes, &kgData)
	userRawKey := kgData.RawKey

	// 7. Verify Data API Request with provisioned Key
	sampleFlight := models.FlightData{
		Aircraft: models.AircraftInfo{Hex: "78091a", Registration: "B-208E"},
		Flight:   models.FlightIdentifier{CallsignICAO: "CSN300"},
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{Latitude: 31.1443, Longitude: 121.8083},
			Altitude: models.AltitudeInfo{BarometricFt: 33000},
			Speed:    models.SpeedInfo{GroundSpeedKts: 480},
		},
		Metadata: models.MetadataInfo{UpstreamSource: "opensky", LastUpdated: time.Now()},
	}
	db.SaveLiveFlights([]models.FlightData{sampleFlight})

	reqData := httptest.NewRequest("GET", "/api/v1/flights", nil)
	reqData.Header.Set("X-API-Key", userRawKey)
	wData := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wData, reqData)

	if wData.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK with valid API key, got %d: %s", wData.Code, wData.Body.String())
	}

	var dataResp models.APIResponse
	json.Unmarshal(wData.Body.Bytes(), &dataResp)
	if dataResp.Attribution == nil || len(dataResp.Attribution.Sources) == 0 {
		t.Errorf("Attribution missing in API response!")
	}

	// 8. Test Field Restrictions: Admin restricts fields for normal users
	// Admin login
	adminLoginPayload := models.LoginRequest{
		Email:    testAdminEmail,
		Password: testAdminPassword,
	}
	bodyBytes, _ = json.Marshal(adminLoginPayload)
	reqAdminLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(bodyBytes))
	reqAdminLogin.Header.Set("Content-Type", "application/json")
	wAdminLogin := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wAdminLogin, reqAdminLogin)

	if wAdminLogin.Code != http.StatusOK {
		t.Fatalf("Admin login failed, got %d: %s", wAdminLogin.Code, wAdminLogin.Body.String())
	}

	var adminLoginResp models.APIResponse
	json.Unmarshal(wAdminLogin.Body.Bytes(), &adminLoginResp)
	admBytes, _ := json.Marshal(adminLoginResp.Data)
	var admData struct {
		Token string `json:"token"`
	}
	json.Unmarshal(admBytes, &admData)
	adminToken := admData.Token

	// Update field restrictions: normal users ONLY get 'flight' and 'telemetry'
	restrictPayload := models.FieldRestrictionsConfig{
		AllowedFields: []string{"flight", "telemetry"},
	}
	bodyBytes, _ = json.Marshal(restrictPayload)
	reqRestrict := httptest.NewRequest("POST", "/api/v1/admin/field-restrictions", bytes.NewReader(bodyBytes))
	reqRestrict.Header.Set("Content-Type", "application/json")
	reqRestrict.Header.Set("Authorization", "Bearer "+adminToken)
	wRestrict := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wRestrict, reqRestrict)

	if wRestrict.Code != http.StatusOK {
		t.Fatalf("Failed to set field restrictions, got %d", wRestrict.Code)
	}

	// Now normal user calls /flights -> 'aircraft' must be omitted, 'flight' present
	reqNormalCheck := httptest.NewRequest("GET", "/api/v1/flights", nil)
	reqNormalCheck.Header.Set("X-API-Key", userRawKey)
	wNormalCheck := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wNormalCheck, reqNormalCheck)

	var normalResp models.APIResponse
	json.Unmarshal(wNormalCheck.Body.Bytes(), &normalResp)
	normalFlightsBytes, _ := json.Marshal(normalResp.Data)
	var normalFlightMaps []map[string]interface{}
	json.Unmarshal(normalFlightsBytes, &normalFlightMaps)

	if len(normalFlightMaps) > 0 {
		if _, hasAircraft := normalFlightMaps[0]["aircraft"]; hasAircraft {
			t.Errorf("Expected 'aircraft' field to be masked for normal user, but it was present")
		}
		if _, hasFlight := normalFlightMaps[0]["flight"]; !hasFlight {
			t.Errorf("Expected 'flight' field to be present for normal user, but it was missing")
		}
	}

	// Admin key calls /flights -> ALL fields must be present (never masked)
	adminUser, _ := db.GetUserByEmail(testAdminEmail)
	adminRawKey, _, _ := db.GenerateAPIKey(adminUser.ID, "Admin Unrestricted Key")

	reqAdminCheck := httptest.NewRequest("GET", "/api/v1/flights", nil)
	reqAdminCheck.Header.Set("X-API-Key", adminRawKey)
	wAdminCheck := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wAdminCheck, reqAdminCheck)

	var adminResp models.APIResponse
	json.Unmarshal(wAdminCheck.Body.Bytes(), &adminResp)
	adminFlightsBytes, _ := json.Marshal(adminResp.Data)
	var adminFlights []models.FlightData
	json.Unmarshal(adminFlightsBytes, &adminFlights)

	if len(adminFlights) > 0 {
		if adminFlights[0].Aircraft.Hex == "" {
			t.Errorf("Expected full unmasked aircraft.hex for admin key, got empty")
		}
	}
}

func TestCapIntegration(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Check /api/v1/auth/cap-config
	req := httptest.NewRequest("GET", "/api/v1/auth/cap-config", nil)
	w := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 from /api/v1/auth/cap-config, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode cap-config response: %v", err)
	}

	capMap, ok := resp.Data.(map[string]interface{})
	if !ok || capMap["site_key"] == "" || capMap["api_endpoint"] == "" {
		t.Fatalf("Expected valid site_key and api_endpoint, got %+v", resp.Data)
	}

	// 2. Test send-code with test Cap token
	sendReq := models.SendCodeRequest{
		Email:    "captest@example.com",
		CapToken: "test-cap-token-ok",
	}
	body, _ := json.Marshal(sendReq)
	reqSend := httptest.NewRequest("POST", "/api/v1/auth/send-code", bytes.NewReader(body))
	reqSend.Header.Set("Content-Type", "application/json")
	wSend := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(wSend, reqSend)

	if wSend.Code != http.StatusOK {
		t.Fatalf("Expected 200 from /api/v1/auth/send-code with valid Cap token, got %d: %s", wSend.Code, wSend.Body.String())
	}
}
