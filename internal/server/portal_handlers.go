package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"contrail/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// handleCapConfig exposes the self-hosted Cap site key and endpoint for client widget initialization.
func (s *Server) handleCapConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	s.respondJSON(w, r, http.StatusOK, map[string]string{
		"site_key":     s.cfg.CapSiteKey,
		"api_endpoint": s.cfg.CapAPIEndpoint,
	}, 1, 1, 1, 1, start)
}

// handleCaptcha provides an anti-bot mathematical visual challenge (legacy/fallback).
func (s *Server) handleCaptcha(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	challenge, err := s.captcha.Generate()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to generate human verification", start)
		return
	}
	s.respondJSON(w, r, http.StatusOK, challenge, 1, 1, 1, 1, start)
}

// handleSendCode verifies the CAPTCHA and dispatches an email verification code.
func (s *Server) handleSendCode(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
		return
	}

	var req models.SendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request payload", start)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		s.respondError(w, http.StatusBadRequest, "Valid email address required", start)
		return
	}

	// Verify human verification (Cap PoW token preferred; fallback to math captcha for test/legacy compatibility)
	if req.CapToken != "" {
		valid, err := s.capVerifier.Verify(req.CapToken)
		if !valid || err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid or expired human verification (Cap challenge)", start)
			return
		}
	} else if req.CaptchaID != "" && s.captcha.Verify(req.CaptchaID, req.CaptchaAnswer) {
		// Legacy / test fallback
	} else if os.Getenv("ENV") == "test" || s.cfg.CapSecretKey == "" {
		// Allowed in unit test environment
	} else {
		s.respondError(w, http.StatusBadRequest, "Human verification (Cap challenge) required", start)
		return
	}

	// Check registration channel status
	if !s.db.IsRegistrationEnabled() {
		// Send registration channel closed notice (without any verification code)
		go func(to string) {
			_ = s.mailer.SendRegistrationClosedNotice(to)
		}(req.Email)

		s.respondJSON(w, r, http.StatusOK, map[string]string{
			"message": "Notice dispatched to your email address",
		}, 1, 1, 1, 1, start)
		return
	}

	// Generate 6-digit cryptographic code
	nBig, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to generate security code", start)
		return
	}
	code := fmt.Sprintf("%06d", nBig.Int64()+100000)

	// Save code to database with 10 minute TTL
	if err := s.db.SaveVerificationCode(req.Email, code, "register", 10*time.Minute); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to store verification record", start)
		return
	}

	// Dispatch email
	go func(to, c string) {
		_ = s.mailer.SendVerificationCode(to, c)
	}(req.Email, code)

	s.respondJSON(w, r, http.StatusOK, map[string]string{
		"message": "Verification code dispatched to your email address",
	}, 1, 1, 1, 1, start)
}

// handleRegister validates the email verification code and creates the user account.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
		return
	}

	if !s.db.IsRegistrationEnabled() {
		s.respondError(w, http.StatusForbidden, "Registration channel is currently closed by administrator", start)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request payload", start)
		return
	}

	if !req.AgreeTerms {
		s.respondError(w, http.StatusBadRequest, "You must agree to the non-commercial terms of service", start)
		return
	}

	// Verify Cap human verification if provided
	if req.CapToken != "" {
		valid, _ := s.capVerifier.Verify(req.CapToken)
		_ = valid // Cap token was verified when dispatching the email code
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !s.db.VerifyCode(req.Email, req.VerificationCode, "register") {
		s.respondError(w, http.StatusBadRequest, "Invalid or expired verification code", start)
		return
	}

	user, err := s.db.CreateUser(req.Name, req.Email, req.Password, req.Purpose, req.AgreeTerms)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error(), start)
		return
	}

	token, _ := s.auth.GenerateJWT(user)

	s.respondJSON(w, r, http.StatusCreated, map[string]interface{}{
		"message": "Registration successful. You are now logged in.",
		"token":   token,
		"user":    user,
	}, 1, 1, 1, 1, start)
}

// handleLogin validates credentials and issues a signed JWT session.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request payload", start)
		return
	}

	// Verify human verification (Cap challenge)
	if req.CapToken != "" {
		valid, err := s.capVerifier.Verify(req.CapToken)
		if !valid || err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid or expired human verification (Cap challenge)", start)
			return
		}
	} else if os.Getenv("ENV") != "test" && s.cfg.CapSecretKey != "" {
		s.respondError(w, http.StatusBadRequest, "Human verification (Cap challenge) required to sign in", start)
		return
	}

	user, err := s.db.GetUserByEmail(req.Email)
	if err != nil || user == nil {
		s.respondError(w, http.StatusUnauthorized, "Invalid email or password", start)
		return
	}

	if user.Status == "banned" {
		s.respondError(w, http.StatusForbidden, "Your account has been suspended by an administrator", start)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.respondError(w, http.StatusUnauthorized, "Invalid email or password", start)
		return
	}

	token, err := s.auth.GenerateJWT(user)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to generate authentication token", start)
		return
	}

	s.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	}, 1, 1, 1, 1, start)
}

// handleMe returns the currently authenticated user profile.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Authentication required", start)
		return
	}
	s.respondJSON(w, r, http.StatusOK, map[string]interface{}{"user": user}, 1, 1, 1, 1, start)
}

// handlePasswordUpdate updates user account password.
func (s *Server) handlePasswordUpdate(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
		return
	}

	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Authentication required", start)
		return
	}

	var req models.UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.NewPassword) < 8 {
		s.respondError(w, http.StatusBadRequest, "New password must be at least 8 characters", start)
		return
	}

	// Verify old password
	fullUser, _ := s.db.GetUserByID(user.ID)
	if fullUser == nil || bcrypt.CompareHashAndPassword([]byte(fullUser.PasswordHash), []byte(req.OldPassword)) != nil {
		s.respondError(w, http.StatusBadRequest, "Incorrect existing password", start)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to hash new password", start)
		return
	}

	if err := s.db.UpdateUserPassword(user.ID, string(newHash)); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to update password", start)
		return
	}

	s.respondJSON(w, r, http.StatusOK, map[string]string{"message": "Password updated successfully"}, 1, 1, 1, 1, start)
}

// handlePortalKeys manages API key listing and creation for the logged-in user.
func (s *Server) handlePortalKeys(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Authentication required", start)
		return
	}

	switch r.Method {
	case http.MethodGet:
		keys, err := s.db.GetUserAPIKeys(user.ID)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to list keys: "+err.Error(), start)
			return
		}
		s.respondJSON(w, r, http.StatusOK, keys, len(keys), len(keys), 1, 1, start)

	case http.MethodPost:
		var req models.CreateAPIKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request payload", start)
			return
		}

		if !req.AgreeTerms {
			s.respondError(w, http.StatusBadRequest, "You must certify non-commercial usage terms", start)
			return
		}

		rawKey, apiKey, err := s.db.GenerateAPIKey(user.ID, req.KeyName)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to generate API key: "+err.Error(), start)
			return
		}

		s.respondJSON(w, r, http.StatusCreated, map[string]interface{}{
			"raw_key": rawKey,
			"key":     apiKey,
		}, 1, 1, 1, 1, start)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
	}
}

// handleRevokeKey revokes an API key.
func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
		return
	}

	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Authentication required", start)
		return
	}

	// Path: /api/v1/portal/keys/{id}/revoke
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		s.respondError(w, http.StatusBadRequest, "Invalid key ID parameter", start)
		return
	}
	keyID, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid key ID", start)
		return
	}

	isAdmin := (user.Role == "admin")
	if err := s.db.RevokeAPIKey(keyID, user.ID, isAdmin); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error(), start)
		return
	}

	s.respondJSON(w, r, http.StatusOK, map[string]string{"message": "API key permanently deleted"}, 1, 1, 1, 1, start)
}

// handlePortalStats returns telemetry and quota for the current user.
func (s *Server) handlePortalStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	user := s.getUserFromContext(r)
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "Authentication required", start)
		return
	}

	keys, _ := s.db.GetUserAPIKeys(user.ID)
	var todayCalls, totalCalls int64
	for _, k := range keys {
		todayCalls += k.TodayRequests
		totalCalls += k.TotalRequests
	}

	data := map[string]interface{}{
		"today_requests": todayCalls,
		"total_requests": totalCalls,
		"daily_quota":    s.cfg.RateLimitDaily,
		"rpm_limit":      s.cfg.RateLimitRPM,
		"rph_limit":      s.cfg.RateLimitRPH,
		"current_rpm":    0,
	}

	s.respondJSON(w, r, http.StatusOK, data, 1, 1, 1, 1, start)
}

// --- Administrator Endpoints ---

// handleAdminUsers lists all registered accounts.
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	user := s.getUserFromContext(r)
	if user == nil || user.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	users, err := s.db.ListAllUsers()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to list users", start)
		return
	}

	s.respondJSON(w, r, http.StatusOK, users, len(users), len(users), 1, 1, start)
}

// handleAdminUserStatus updates user status (active/banned).
func (s *Server) handleAdminUserStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
		return
	}

	adminUser := s.getUserFromContext(r)
	if adminUser == nil || adminUser.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	// Path: /api/v1/admin/users/{id}/status
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		s.respondError(w, http.StatusBadRequest, "Invalid user ID", start)
		return
	}
	targetUserID, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid user ID", start)
		return
	}

	var req models.UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Status != "active" && req.Status != "banned") {
		s.respondError(w, http.StatusBadRequest, "Status must be 'active' or 'banned'", start)
		return
	}

	if err := s.db.UpdateUserStatus(targetUserID, req.Status); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to update status", start)
		return
	}

	s.respondJSON(w, r, http.StatusOK, map[string]string{"message": "User status updated to " + req.Status}, 1, 1, 1, 1, start)
}

// handleAdminDeleteUser permanently deletes a user and associated keys.
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodDelete {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
		return
	}

	adminUser := s.getUserFromContext(r)
	if adminUser == nil || adminUser.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		s.respondError(w, http.StatusBadRequest, "Invalid user ID", start)
		return
	}
	targetUserID, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid user ID", start)
		return
	}

	if targetUserID == adminUser.ID {
		s.respondError(w, http.StatusBadRequest, "Cannot delete master administrator account", start)
		return
	}

	if err := s.db.DeleteUser(targetUserID); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to delete user", start)
		return
	}

	s.respondJSON(w, r, http.StatusOK, map[string]string{"message": "User deleted successfully"}, 1, 1, 1, 1, start)
}

// handleAdminFieldRestrictions configures which fields normal users receive.
func (s *Server) handleAdminFieldRestrictions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	adminUser := s.getUserFromContext(r)
	if adminUser == nil || adminUser.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	switch r.Method {
	case http.MethodGet:
		allowed := s.db.GetAllowedFieldsForNormalUsers()
		s.respondJSON(w, r, http.StatusOK, map[string]interface{}{
			"allowed_fields": allowed,
			"notice":         "Admin keys bypass all field filtering and always return 100% full fields.",
		}, 1, 1, 1, 1, start)

	case http.MethodPost:
		var req models.FieldRestrictionsConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid payload", start)
			return
		}
		if len(req.AllowedFields) == 0 {
			s.respondError(w, http.StatusBadRequest, "At least one allowed field must be selected", start)
			return
		}
		if err := s.db.SetAllowedFieldsForNormalUsers(req.AllowedFields); err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to save restrictions: "+err.Error(), start)
			return
		}
		s.respondJSON(w, r, http.StatusOK, map[string]interface{}{
			"message":        "Field restrictions updated successfully",
			"allowed_fields": req.AllowedFields,
		}, 1, 1, 1, 1, start)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
	}
}

// handleAdminRegistrationStatus queries or updates the registration channel status.
func (s *Server) handleAdminRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	adminUser := s.getUserFromContext(r)
	if adminUser == nil || adminUser.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	switch r.Method {
	case http.MethodGet:
		enabled := s.db.IsRegistrationEnabled()
		s.respondJSON(w, r, http.StatusOK, map[string]interface{}{
			"registration_enabled": enabled,
		}, 1, 1, 1, 1, start)

	case http.MethodPost:
		var req struct {
			RegistrationEnabled bool `json:"registration_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid payload", start)
			return
		}
		if err := s.db.SetRegistrationEnabled(req.RegistrationEnabled); err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to update registration status: "+err.Error(), start)
			return
		}
		s.respondJSON(w, r, http.StatusOK, map[string]interface{}{
			"message":              "Registration status updated successfully",
			"registration_enabled": req.RegistrationEnabled,
		}, 1, 1, 1, 1, start)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed", start)
	}
}

// handleAdminAlerts returns rate limit violation alerts sent to users.
func (s *Server) handleAdminAlerts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	adminUser := s.getUserFromContext(r)
	if adminUser == nil || adminUser.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	rows, err := s.db.Query(`
		SELECT a.id, a.user_id, a.key_id, a.reason, a.sent_at, u.email, COALESCE(k.key_prefix, 'unknown')
		FROM rate_limit_alerts a
		JOIN users u ON a.user_id = u.id
		LEFT JOIN api_keys k ON a.key_id = k.id
		ORDER BY a.sent_at DESC LIMIT 100
	`)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query alerts", start)
		return
	}
	defer rows.Close()

	type alertRow struct {
		ID        int64  `json:"id"`
		UserID    int64  `json:"user_id"`
		KeyID     int64  `json:"key_id"`
		Reason    string `json:"reason"`
		SentAt    int64  `json:"sent_at"`
		Email     string `json:"email"`
		KeyPrefix string `json:"key_prefix"`
	}

	var alerts []alertRow
	for rows.Next() {
		var a alertRow
		if err := rows.Scan(&a.ID, &a.UserID, &a.KeyID, &a.Reason, &a.SentAt, &a.Email, &a.KeyPrefix); err == nil {
			alerts = append(alerts, a)
		}
	}

	s.respondJSON(w, r, http.StatusOK, alerts, len(alerts), len(alerts), 1, 1, start)
}

// handleAdminUpstreamLogs returns recent upstream ingestion log entries.
func (s *Server) handleAdminUpstreamLogs(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	adminUser := s.getUserFromContext(r)
	if adminUser == nil || adminUser.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	providerFilter := r.URL.Query().Get("provider")
	hubFilter := r.URL.Query().Get("hub")
	statusFilter := r.URL.Query().Get("status")

	logs, err := s.db.GetUpstreamFetchLogs(limit, providerFilter, hubFilter, statusFilter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query upstream logs: "+err.Error(), start)
		return
	}

	s.respondJSON(w, r, http.StatusOK, logs, len(logs), len(logs), 1, 1, start)
}

// handleAdminUpstreamStatus returns real-time status of all configured upstream telemetry providers.
func (s *Server) handleAdminUpstreamStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	adminUser := s.getUserFromContext(r)
	if adminUser == nil || adminUser.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	statusMap := map[string]interface{}{}
	if s.poller != nil {
		status := s.poller.GetStatus()
		if ups, ok := status["upstreams"]; ok {
			statusMap["upstreams"] = ups
		}
	}

	if s.db != nil {
		meanKm, p95Km, samples, _ := s.db.GetExtrapolationAccuracyStats(24)
		statusMap["accuracy_stats_24h"] = map[string]interface{}{
			"mean_displacement_error_km": meanKm,
			"p95_displacement_error_km":  p95Km,
			"total_calibrated_samples":   samples,
		}
	}

	s.respondJSON(w, r, http.StatusOK, statusMap, 1, 1, 1, 1, start)
}

// handleAdminSendTestAlert sends an alert email to verify the alert notification pathway.
func (s *Server) handleAdminSendTestAlert(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "POST method required", start)
		return
	}

	adminUser := s.getUserFromContext(r)
	if adminUser == nil || adminUser.Role != "admin" {
		s.respondError(w, http.StatusForbidden, "Administrator privileges required", start)
		return
	}

	var req struct {
		Recipient string `json:"recipient"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	recipient := strings.TrimSpace(req.Recipient)
	if recipient == "" {
		recipient = s.cfg.AlertNotifyEmail
	}
	if recipient == "" {
		recipient = s.cfg.AlertNotifyEmail
	}

	subject := "Contrail Telemetry Alert Test"
	content := fmt.Sprintf("This is an operational test alert dispatched from Contrail Admin Console.\nTimestamp: %s\nTarget: %s\nStatus: Operational",
		time.Now().UTC().Format(time.RFC3339), recipient)

	err := s.mailer.SendStatusAlert(recipient, subject, content)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to send email alert: "+err.Error(), start)
		return
	}

	s.respondJSON(w, r, http.StatusOK, map[string]string{
		"recipient": recipient,
		"status":    "sent",
		"message":   "Test alert email successfully dispatched",
	}, 1, 1, 1, 1, start)
}


// Helper: extract authenticated user from Bearer JWT in request header.
func (s *Server) getUserFromContext(r *http.Request) *models.User {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := s.auth.VerifyJWT(tokenStr)
	if err != nil || claims == nil {
		return nil
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil || user == nil || user.Status == "banned" {
		return nil
	}
	return user
}
