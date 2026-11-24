package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"contrail/internal/config"
	"contrail/internal/database"
	"contrail/internal/mailer"
	"contrail/internal/models"
)

// JWTClaims holds authenticated session data.
type JWTClaims struct {
	UserID    int64  `json:"uid"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"exp"`
}

type rateWindow struct {
	minuteBucket map[int64]int // epoch minute -> count
	hourBucket   map[int64]int // epoch hour -> count
	alertCount   int
	lastAlertAt  int64
}

// AuthManager handles authentication, JWT sessions, rate limiting, and alert dispatch.
type AuthManager struct {
	cfg     *config.Config
	db      *database.DB
	mailer  *mailer.Mailer
	mu      sync.Mutex
	windows map[int64]*rateWindow // key_id -> rateWindow
}

// NewAuthManager initializes the AuthManager.
func NewAuthManager(cfg *config.Config, db *database.DB, m *mailer.Mailer) *AuthManager {
	am := &AuthManager{
		cfg:     cfg,
		db:      db,
		mailer:  m,
		windows: make(map[int64]*rateWindow),
	}

	// Periodic cleanup of stale rate buckets
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			am.cleanupWindows()
		}
	}()

	return am
}

func (am *AuthManager) cleanupWindows() {
	am.mu.Lock()
	defer am.mu.Unlock()

	nowMinute := time.Now().Unix() / 60
	nowHour := time.Now().Unix() / 3600

	for _, w := range am.windows {
		for m := range w.minuteBucket {
			if nowMinute-m > 5 {
				delete(w.minuteBucket, m)
			}
		}
		for h := range w.hourBucket {
			if nowHour-h > 2 {
				delete(w.hourBucket, h)
			}
		}
	}
}

// GenerateJWT creates a signed JWT string.
func (am *AuthManager) GenerateJWT(user *models.User) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claims := JWTClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)

	sig := am.signHMAC(header + "." + payload)
	return header + "." + payload + "." + sig, nil
}

// VerifyJWT parses and verifies a JWT token string.
func (am *AuthManager) VerifyJWT(tokenStr string) (*JWTClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	headerPayload := parts[0] + "." + parts[1]
	expectedSig := am.signHMAC(headerPayload)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, errors.New("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("failed to decode claims")
	}

	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, errors.New("failed to parse claims")
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

func (am *AuthManager) signHMAC(data string) string {
	h := hmac.New(sha256.New, []byte(am.cfg.JWTSecret))
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// RateLimitStatus holds detailed rate limiting status for an API key.
type RateLimitStatus struct {
	Allowed         bool   `json:"allowed"`
	LimitRPM        int    `json:"limit_rpm"`
	RemainingRPM    int    `json:"remaining_rpm"`
	ResetEpoch      int64  `json:"reset_epoch"`
	LimitDaily      int    `json:"limit_daily"`
	RemainingDaily  int    `json:"remaining_daily"`
	DailyResetEpoch int64  `json:"daily_reset_epoch"`
	ErrorMessage    string `json:"error_message,omitempty"`
	IsAdmin         bool   `json:"is_admin"`
}

// EvaluateRateLimit calculates precise RPM, Hourly, and Daily quotas and returns rate limit headers.
func (am *AuthManager) EvaluateRateLimit(key *models.APIKey, user *models.User) RateLimitStatus {
	now := time.Now().UTC()
	nowUnix := now.Unix()
	nextMinuteEpoch := (nowUnix/60 + 1) * 60
	nextMidnightEpoch := now.Truncate(24 * time.Hour).Add(24 * time.Hour).Unix()

	limitDaily := am.cfg.RateLimitDaily
	if limitDaily <= 0 {
		limitDaily = 10000
	}
	limitRPM := am.cfg.RateLimitRPM
	if limitRPM <= 0 {
		limitRPM = 60
	}

	// Administrator keys bypass rate limits completely
	if user != nil && user.Role == "admin" {
		return RateLimitStatus{
			Allowed:         true,
			LimitRPM:        limitRPM,
			RemainingRPM:    limitRPM,
			ResetEpoch:      nextMinuteEpoch,
			LimitDaily:      limitDaily,
			RemainingDaily:  limitDaily,
			DailyResetEpoch: nextMidnightEpoch,
			IsAdmin:         true,
		}
	}

	remDaily := limitDaily - int(key.TodayRequests)
	if remDaily < 0 {
		remDaily = 0
	}

	// Daily quota check
	if key.TodayRequests >= int64(limitDaily) {
		return RateLimitStatus{
			Allowed:         false,
			LimitRPM:        limitRPM,
			RemainingRPM:    0,
			ResetEpoch:      nextMinuteEpoch,
			LimitDaily:      limitDaily,
			RemainingDaily:  0,
			DailyResetEpoch: nextMidnightEpoch,
			ErrorMessage:    fmt.Sprintf("Daily quota exceeded (%d requests/day limit). Quota resets at 00:00 UTC.", limitDaily),
		}
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	nowMinute := nowUnix / 60
	nowHour := nowUnix / 3600

	w, exists := am.windows[key.ID]
	if !exists {
		w = &rateWindow{
			minuteBucket: make(map[int64]int),
			hourBucket:   make(map[int64]int),
		}
		am.windows[key.ID] = w
	}

	w.minuteBucket[nowMinute]++
	w.hourBucket[nowHour]++

	currentRPM := w.minuteBucket[nowMinute]
	currentRPH := w.hourBucket[nowHour]

	remRPM := limitRPM - currentRPM
	if remRPM < 0 {
		remRPM = 0
	}

	// Check RPM
	if currentRPM > limitRPM {
		w.alertCount++
		if w.alertCount >= 5 && (nowUnix-w.lastAlertAt > 6*3600) {
			w.lastAlertAt = nowUnix
			w.alertCount = 0
			go func(email, prefix string, rpm int, uID, kID int64) {
				_ = am.mailer.SendRateLimitAlert(email, prefix, rpm)
				_, _ = am.db.Exec(`
					INSERT INTO rate_limit_alerts (user_id, key_id, reason, sent_at)
					VALUES (?, ?, ?, ?)
				`, uID, kID, fmt.Sprintf("Exceeded RPM (%d > %d)", rpm, am.cfg.RateLimitRPM), time.Now().Unix())
			}(user.Email, key.KeyPrefix, currentRPM, user.ID, key.ID)
		}

		return RateLimitStatus{
			Allowed:         false,
			LimitRPM:        limitRPM,
			RemainingRPM:    0,
			ResetEpoch:      nextMinuteEpoch,
			LimitDaily:      limitDaily,
			RemainingDaily:  remDaily,
			DailyResetEpoch: nextMidnightEpoch,
			ErrorMessage:    fmt.Sprintf("Rate limit exceeded: %d req/min (Limit: %d req/min). Please slow down.", currentRPM, limitRPM),
		}
	}

	// Check RPH
	limitRPH := am.cfg.RateLimitRPH
	if limitRPH > 0 && currentRPH > limitRPH {
		return RateLimitStatus{
			Allowed:         false,
			LimitRPM:        limitRPM,
			RemainingRPM:    remRPM,
			ResetEpoch:      nextMinuteEpoch,
			LimitDaily:      limitDaily,
			RemainingDaily:  remDaily,
			DailyResetEpoch: nextMidnightEpoch,
			ErrorMessage:    fmt.Sprintf("Hourly rate limit exceeded: %d req/hr (Limit: %d req/hr).", currentRPH, limitRPH),
		}
	}

	return RateLimitStatus{
		Allowed:         true,
		LimitRPM:        limitRPM,
		RemainingRPM:    remRPM,
		ResetEpoch:      nextMinuteEpoch,
		LimitDaily:      limitDaily,
		RemainingDaily:  remDaily,
		DailyResetEpoch: nextMidnightEpoch,
	}
}

// CheckRateLimit evaluates RPM and Daily limits.
// Returns allowed bool, currentRPM int, error message if any.
func (am *AuthManager) CheckRateLimit(key *models.APIKey, user *models.User) (bool, int, string) {
	st := am.EvaluateRateLimit(key, user)
	return st.Allowed, st.LimitRPM - st.RemainingRPM, st.ErrorMessage
}

// GetAllowedFieldsMap returns a lowercase lookup map of allowed fields for regular API users.
func (am *AuthManager) GetAllowedFieldsMap() map[string]bool {
	allowedList := am.db.GetAllowedFieldsForNormalUsers()
	allowedMap := make(map[string]bool)
	for _, f := range allowedList {
		allowedMap[strings.ToLower(strings.TrimSpace(f))] = true
	}
	return allowedMap
}

// ApplyFieldFiltering filters FlightData fields based on allowed configuration.
// Admin users always receive 100% of all fields.
func (am *AuthManager) ApplyFieldFiltering(flights []models.FlightData, user *models.User) interface{} {
	if user != nil && user.Role == "admin" {
		return flights // Admin gets all fields unfiltered
	}

	allowedMap := am.GetAllowedFieldsMap()

	// Map each flight to dynamic filtered map
	var filtered []map[string]interface{}
	for _, f := range flights {
		m := make(map[string]interface{})
		if allowedMap["aircraft"] {
			m["aircraft"] = f.Aircraft
		}
		if allowedMap["flight"] {
			m["flight"] = f.Flight
		}
		if allowedMap["route"] {
			m["route"] = f.Route
		}
		if allowedMap["telemetry"] {
			m["telemetry"] = f.Telemetry
		}
		if allowedMap["emergency"] {
			m["emergency"] = f.Emergency
		}
		if allowedMap["diversion_analysis"] {
			m["diversion_analysis"] = f.DiversionAnalysis
		}
		if allowedMap["metadata"] {
			m["metadata"] = f.Metadata
		}
		filtered = append(filtered, m)
	}

	return filtered
}

// ApplySingleFlightFiltering filters a single FlightData record.
func (am *AuthManager) ApplySingleFlightFiltering(f *models.FlightData, user *models.User) interface{} {
	if user != nil && user.Role == "admin" {
		return f
	}

	allowedMap := am.GetAllowedFieldsMap()

	m := make(map[string]interface{})
	if allowedMap["aircraft"] {
		m["aircraft"] = f.Aircraft
	}
	if allowedMap["flight"] {
		m["flight"] = f.Flight
	}
	if allowedMap["route"] {
		m["route"] = f.Route
	}
	if allowedMap["telemetry"] {
		m["telemetry"] = f.Telemetry
	}
	if allowedMap["emergency"] {
		m["emergency"] = f.Emergency
	}
	if allowedMap["diversion_analysis"] {
		m["diversion_analysis"] = f.DiversionAnalysis
	}
	if allowedMap["metadata"] {
		m["metadata"] = f.Metadata
	}
	return m
}
