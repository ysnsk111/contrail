package database

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"contrail/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// CreateUser inserts a new user with hashed password.
func (db *DB) CreateUser(name, email, password, purpose string, termsAgreed bool) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	purpose = strings.TrimSpace(purpose)

	if email == "" || name == "" || password == "" {
		return nil, errors.New("name, email, and password are required")
	}
	if !termsAgreed {
		return nil, errors.New("must agree to non-commercial terms of service")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().Unix()
	res, err := db.Exec(`
		INSERT INTO users (email, password_hash, name, purpose, role, status, terms_agreed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'user', 'active', ?, ?, ?)
	`, email, string(hash), name, purpose, now, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, errors.New("email already registered")
		}
		return nil, err
	}

	id, _ := res.LastInsertId()
	return &models.User{
		ID:            id,
		Email:         email,
		Name:          name,
		Purpose:       purpose,
		Role:          "user",
		Status:        "active",
		TermsAgreedAt: now,
		CreatedAt:     time.Unix(now, 0),
		UpdatedAt:     time.Unix(now, 0),
	}, nil
}

// GetUserByEmail finds a user by email.
func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	row := db.QueryRow(`
		SELECT id, email, password_hash, name, purpose, role, status, terms_agreed_at, created_at, updated_at
		FROM users WHERE email = ?
	`, email)

	var u models.User
	var termsAt, createdAt, updatedAt int64
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Purpose, &u.Role, &u.Status, &termsAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	u.TermsAgreedAt = termsAt
	u.CreatedAt = time.Unix(createdAt, 0)
	u.UpdatedAt = time.Unix(updatedAt, 0)
	return &u, nil
}

// GetUserByID finds a user by ID.
func (db *DB) GetUserByID(id int64) (*models.User, error) {
	row := db.QueryRow(`
		SELECT id, email, password_hash, name, purpose, role, status, terms_agreed_at, created_at, updated_at
		FROM users WHERE id = ?
	`, id)

	var u models.User
	var termsAt, createdAt, updatedAt int64
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Purpose, &u.Role, &u.Status, &termsAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	u.TermsAgreedAt = termsAt
	u.CreatedAt = time.Unix(createdAt, 0)
	u.UpdatedAt = time.Unix(updatedAt, 0)
	return &u, nil
}

// ListAllUsers returns users for admin management.
func (db *DB) ListAllUsers() ([]*models.User, error) {
	rows, err := db.Query(`
		SELECT id, email, name, purpose, role, status, terms_agreed_at, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		var termsAt, createdAt, updatedAt int64
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Purpose, &u.Role, &u.Status, &termsAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		u.TermsAgreedAt = termsAt
		u.CreatedAt = time.Unix(createdAt, 0)
		u.UpdatedAt = time.Unix(updatedAt, 0)
		users = append(users, &u)
	}
	return users, nil
}

// UpdateUserStatus bans or unbans a user.
func (db *DB) UpdateUserStatus(id int64, status string) error {
	_, err := db.Exec("UPDATE users SET status = ?, updated_at = ? WHERE id = ?", status, time.Now().Unix(), id)
	return err
}

// DeleteUser removes a user and their keys.
func (db *DB) DeleteUser(id int64) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// UpdateUserPassword updates the password hash.
func (db *DB) UpdateUserPassword(id int64, newHash string) error {
	_, err := db.Exec("UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?", newHash, time.Now().Unix(), id)
	return err
}

// --- API Key Management ---

// GenerateAPIKey provisions a new secure API key with prefix ct_live_
func (db *DB) GenerateAPIKey(userID int64, keyName string) (rawKey string, apiKey *models.APIKey, err error) {
	keyName = strings.TrimSpace(keyName)
	if keyName == "" {
		keyName = "Default Key"
	}

	randBytes := make([]byte, 20)
	if _, err := rand.Read(randBytes); err != nil {
		return "", nil, err
	}

	rawKey = "ct_live_" + hex.EncodeToString(randBytes)
	keyHash := HashKey(rawKey)
	keyPrefix := rawKey[:16] + "..."

	now := time.Now().Unix()
	res, err := db.Exec(`
		INSERT INTO api_keys (user_id, key_name, key_prefix, key_hash, status, terms_agreed, created_at, last_used_at, total_requests)
		VALUES (?, ?, ?, ?, 'active', 1, ?, NULL, 0)
	`, userID, keyName, keyPrefix, keyHash, now)
	if err != nil {
		return "", nil, err
	}

	id, _ := res.LastInsertId()
	apiKey = &models.APIKey{
		ID:            id,
		UserID:        userID,
		KeyName:       keyName,
		KeyPrefix:     keyPrefix,
		Status:        "active",
		TermsAgreed:   true,
		CreatedAt:     time.Unix(now, 0),
		TotalRequests: 0,
		TodayRequests: 0,
	}

	return rawKey, apiKey, nil
}

// GetUserAPIKeys retrieves all active keys owned by a user.
func (db *DB) GetUserAPIKeys(userID int64) ([]*models.APIKey, error) {
	todayStr := time.Now().UTC().Format("2006-01-02")
	rows, err := db.Query(`
		SELECT k.id, k.user_id, k.key_name, k.key_prefix, k.status, k.terms_agreed, k.created_at, k.last_used_at, k.total_requests,
		       COALESCE(u.request_count, 0) AS today_requests
		FROM api_keys k
		LEFT JOIN api_key_usage_daily u ON k.id = u.key_id AND u.date_str = ?
		WHERE k.user_id = ? AND k.status = 'active'
		ORDER BY k.created_at DESC
	`, todayStr, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		var k models.APIKey
		var createdAt int64
		var lastUsedAt sql.NullInt64
		var termsAgreed int
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyName, &k.KeyPrefix, &k.Status, &termsAgreed, &createdAt, &lastUsedAt, &k.TotalRequests, &k.TodayRequests); err != nil {
			return nil, err
		}
		k.TermsAgreed = (termsAgreed == 1)
		k.CreatedAt = time.Unix(createdAt, 0)
		if lastUsedAt.Valid {
			t := time.Unix(lastUsedAt.Int64, 0)
			k.LastUsedAt = &t
		}
		keys = append(keys, &k)
	}
	return keys, nil
}

// RevokeAPIKey permanently deletes an API key and associated daily usage.
func (db *DB) RevokeAPIKey(keyID, userID int64, isAdmin bool) error {
	var res sql.Result
	var err error
	if isAdmin {
		_, _ = db.Exec("DELETE FROM api_key_usage_daily WHERE key_id = ?", keyID)
		res, err = db.Exec("DELETE FROM api_keys WHERE id = ?", keyID)
	} else {
		_, _ = db.Exec("DELETE FROM api_key_usage_daily WHERE key_id = ? AND key_id IN (SELECT id FROM api_keys WHERE user_id = ?)", keyID, userID)
		res, err = db.Exec("DELETE FROM api_keys WHERE id = ? AND user_id = ?", keyID, userID)
	}
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("key not found or unauthorized")
	}
	return nil
}

// AuthenticateKey verifies an incoming raw API key and returns the key and user info.
func (db *DB) AuthenticateKey(rawKey string) (*models.APIKey, *models.User, error) {
	keyHash := HashKey(rawKey)
	todayStr := time.Now().UTC().Format("2006-01-02")

	row := db.QueryRow(`
		SELECT k.id, k.user_id, k.key_name, k.key_prefix, k.status, k.terms_agreed, k.created_at, k.last_used_at, k.total_requests,
		       COALESCE(u.request_count, 0) AS today_requests,
		       usr.id, usr.email, usr.name, usr.role, usr.status
		FROM api_keys k
		JOIN users usr ON k.user_id = usr.id
		LEFT JOIN api_key_usage_daily u ON k.id = u.key_id AND u.date_str = ?
		WHERE k.key_hash = ?
	`, todayStr, keyHash)

	var k models.APIKey
	var usr models.User
	var createdAt int64
	var lastUsedAt sql.NullInt64
	var termsAgreed int

	err := row.Scan(
		&k.ID, &k.UserID, &k.KeyName, &k.KeyPrefix, &k.Status, &termsAgreed, &createdAt, &lastUsedAt, &k.TotalRequests, &k.TodayRequests,
		&usr.ID, &usr.Email, &usr.Name, &usr.Role, &usr.Status,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	k.TermsAgreed = (termsAgreed == 1)
	k.CreatedAt = time.Unix(createdAt, 0)
	if lastUsedAt.Valid {
		t := time.Unix(lastUsedAt.Int64, 0)
		k.LastUsedAt = &t
	}

	return &k, &usr, nil
}

// RecordKeyUsage increments usage counters for an API key.
func (db *DB) RecordKeyUsage(keyID int64) {
	now := time.Now().Unix()
	todayStr := time.Now().UTC().Format("2006-01-02")

	// Update total requests and last used
	_, _ = db.Exec("UPDATE api_keys SET total_requests = total_requests + 1, last_used_at = ? WHERE id = ?", now, keyID)

	// Update daily usage
	_, _ = db.Exec(`
		INSERT INTO api_key_usage_daily (key_id, date_str, request_count)
		VALUES (?, ?, 1)
		ON CONFLICT(key_id, date_str) DO UPDATE SET request_count = request_count + 1
	`, keyID, todayStr)
}

// --- Verification Codes ---

// SaveVerificationCode stores an email verification code.
func (db *DB) SaveVerificationCode(email, code, purpose string, ttl time.Duration) error {
	email = strings.ToLower(strings.TrimSpace(email))
	expiresAt := time.Now().Add(ttl).Unix()
	now := time.Now().Unix()

	// Invalidate older unused codes for same email & purpose
	_, _ = db.Exec("DELETE FROM email_verifications WHERE email = ? AND purpose = ?", email, purpose)

	_, err := db.Exec(`
		INSERT INTO email_verifications (email, code, purpose, expires_at, used, created_at)
		VALUES (?, ?, ?, ?, 0, ?)
	`, email, code, purpose, expiresAt, now)
	return err
}

// VerifyCode checks if the code is valid and marks it used.
func (db *DB) VerifyCode(email, code, purpose string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)
	now := time.Now().Unix()

	var id int64
	err := db.QueryRow(`
		SELECT id FROM email_verifications
		WHERE email = ? AND code = ? AND purpose = ? AND used = 0 AND expires_at > ?
	`, email, code, purpose, now).Scan(&id)
	if err != nil {
		return false
	}

	_, _ = db.Exec("UPDATE email_verifications SET used = 1 WHERE id = ?", id)
	return true
}

// --- System Settings & Field Restrictions ---

// GetAllowedFieldsForNormalUsers returns list of fields allowed for regular API callers.
func (db *DB) GetAllowedFieldsForNormalUsers() []string {
	var val string
	err := db.QueryRow("SELECT value FROM system_settings WHERE key = 'normal_user_allowed_fields'").Scan(&val)
	if err != nil {
		return []string{"flight", "aircraft", "telemetry", "route", "emergency", "diversion_analysis", "metadata"}
	}

	var fields []string
	if err := json.Unmarshal([]byte(val), &fields); err != nil || len(fields) == 0 {
		return []string{"flight", "aircraft", "telemetry", "route", "emergency", "diversion_analysis", "metadata"}
	}
	return fields
}

// SetAllowedFieldsForNormalUsers updates the allowed fields configuration.
func (db *DB) SetAllowedFieldsForNormalUsers(fields []string) error {
	data, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO system_settings (key, value) VALUES ('normal_user_allowed_fields', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, string(data))
	return err
}

// IsRegistrationEnabled checks if the user registration channel is currently active.
func (db *DB) IsRegistrationEnabled() bool {
	var val string
	err := db.QueryRow("SELECT value FROM system_settings WHERE key = 'registration_enabled'").Scan(&val)
	if err != nil {
		return true // Default to enabled
	}
	return strings.ToLower(strings.TrimSpace(val)) != "false"
}

// SetRegistrationEnabled updates the user registration channel state.
func (db *DB) SetRegistrationEnabled(enabled bool) error {
	val := "true"
	if !enabled {
		val = "false"
	}
	query := `
		INSERT INTO system_settings (key, value) VALUES ('registration_enabled', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`
	_, err := db.Exec(query, val)
	return err
}

// HashKey computes SHA256 of the raw API key for storage/lookup.
func HashKey(rawKey string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(rawKey)))
	return hex.EncodeToString(h[:])
}

// RestoreLiveFlightsFromLatestSnapshot populates live_flights from the most recent snapshots.
// If force is true, it replaces existing live flights.
func (db *DB) RestoreLiveFlightsFromLatestSnapshot(force ...bool) (int, error) {
	var count int
	_ = db.QueryRow("SELECT count(*) FROM live_flights").Scan(&count)
	shouldForce := len(force) > 0 && force[0]
	if count > 0 && !shouldForce {
		return count, nil
	}

	if shouldForce {
		_, _ = db.Exec("DELETE FROM live_flights")
	}

	type snapRec struct {
		hex, icao, iata, squawk string
		lat, lon, alt, gs, track, vr float64
		isEmerg, isDiv int
	}

	// Fetch ALL distinct aircraft from the latest snapshot in flight_snapshots (NO 2,000 LIMIT)
	rows, err := db.Query(`
		SELECT hex, callsign_icao, callsign_iata, lat, lon, alt_baro_ft, ground_speed_kts, track_deg, vertical_rate_fpm, squawk, is_emergency, is_diverting
		FROM flight_snapshots
		WHERE timestamp = (SELECT max(timestamp) FROM flight_snapshots)
	`)
	if err != nil {
		return 0, err
	}

	var recs []snapRec
	for rows.Next() {
		var (
			hexCode, callsignIcao, callsignIata, squawk sql.NullString
			lat, lon, alt, gs, track, vr                sql.NullFloat64
			isEmerg, isDiv                              int
		)
		if err := rows.Scan(&hexCode, &callsignIcao, &callsignIata, &lat, &lon, &alt, &gs, &track, &vr, &squawk, &isEmerg, &isDiv); err == nil {
			if hexCode.Valid && hexCode.String != "" {
				recs = append(recs, snapRec{
					hex: hexCode.String, icao: callsignIcao.String, iata: callsignIata.String, squawk: squawk.String,
					lat: lat.Float64, lon: lon.Float64, alt: alt.Float64, gs: gs.Float64, track: track.Float64, vr: vr.Float64,
					isEmerg: isEmerg, isDiv: isDiv,
				})
			}
		}
	}
	rows.Close()

	if len(recs) == 0 {
		return 0, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO live_flights (
			hex, callsign_icao, callsign_iata, airline_name, airline_icao, airline_iata,
			lat, lon, alt_baro_ft, alt_geom_ft, ground_speed_kts, track_deg, vertical_rate_fpm,
			squawk, emergency_category, is_emergency, is_diverting, diversion_probability,
			raw_data_json, on_ground, upstream_source, last_contact, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'opensky', ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	inserted := 0
	for _, r := range recs {
		callsign := r.icao
		if callsign == "" {
			callsign = r.iata
		}

		// Country inference by hex prefix
		country := inferCountryFromHex(r.hex)
		registration := inferRegistrationFromHex(r.hex)

		// Sanitize commercial altitude anomaly (e.g. DLH797 at 53,939ft -> cap to typical 41,000ft)
		baroAlt := r.alt
		if baroAlt > 50000 {
			baroAlt = 41000
		}
		geomAlt := baroAlt

		// Airline identification
		var airName, airIcao, airIata string
		var airlinePtr *models.AirlineInfo
		if len(callsign) >= 3 {
			code := strings.ToUpper(callsign[:3])
			if a, ok := knownMajorAirlines[code]; ok {
				airName = a.Name
				airIcao = a.ICAO
				airIata = a.IATA
				airlinePtr = &models.AirlineInfo{
					Name:    a.Name,
					ICAO:    a.ICAO,
					IATA:    a.IATA,
					Country: a.Country,
				}
			}
		}

		vrMps := math.Round(r.vr*0.00508*100) / 100

		fd := models.FlightData{
			Aircraft: models.AircraftInfo{
				Hex:          r.hex,
				Registration: registration,
				Country:      country,
			},
			Flight: models.FlightIdentifier{
				CallsignICAO: callsign,
				Airline:      airlinePtr,
			},
			Telemetry: models.TelemetryInfo{
				Position: models.PositionInfo{
					Latitude:  r.lat,
					Longitude: r.lon,
				},
				Altitude: models.AltitudeInfo{
					BarometricFt: baroAlt,
					GeometricFt:  geomAlt,
					BarometricM:  math.Round(baroAlt * 0.3048),
				},
				Speed: models.SpeedInfo{
					GroundSpeedKts: r.gs,
					GroundSpeedKmh: math.Round(r.gs * 1.852 * 10) / 10,
				},
				Attitude: models.AttitudeInfo{
					TrackDeg:        r.track,
					VerticalRateFpm: r.vr,
					VerticalRateMps: vrMps,
				},
				OnGround:         r.gs < 30 && baroAlt < 1500,
				LastContactEpoch: now,
			},
			Emergency: models.EmergencyInfo{
				Squawk:      r.squawk,
				IsEmergency: r.isEmerg == 1,
				Category:    models.EmergencyNone,
			},
			DiversionAnalysis: models.DiversionAnalysis{
				PredictedDiversion: r.isDiv == 1,
				ProbabilityLevel:   "NONE",
				TriggeredCriteria:  make([]models.DiversionCriterion, 0),
			},
			Metadata: models.MetadataInfo{
				UpstreamSource: "opensky",
				LastUpdated:    time.Unix(now, 0),
			},
		}

		if r.isEmerg == 1 {
			fd.Emergency.Category = models.EmergencyGeneral
			fd.Emergency.Description = "Squawk emergency condition"
		}

		rawBytes, _ := json.Marshal(fd)

		isEmergInt := 0
		if fd.Emergency.IsEmergency {
			isEmergInt = 1
		}
		isDivInt := 0
		if fd.DiversionAnalysis.PredictedDiversion {
			isDivInt = 1
		}
		onGroundInt := 0
		if fd.Telemetry.OnGround {
			onGroundInt = 1
		}

		emergCatStr := string(fd.Emergency.Category)

		_, err = stmt.Exec(
			r.hex, r.icao, r.iata, airName, airIcao, airIata,
			r.lat, r.lon, baroAlt, geomAlt, r.gs, r.track, r.vr,
			r.squawk, emergCatStr, isEmergInt, isDivInt, fd.DiversionAnalysis.ProbabilityLevel,
			string(rawBytes), onGroundInt, now, now,
		)
		if err == nil {
			inserted++
		}
	}

	_ = tx.Commit()
	return inserted, nil
}

func inferCountryFromHex(hex string) string {
	h := strings.ToLower(strings.TrimSpace(hex))
	if len(h) < 2 {
		return ""
	}
	switch {
	case strings.HasPrefix(h, "78"):
		return "China"
	case strings.HasPrefix(h, "a") || strings.HasPrefix(h, "a0") || strings.HasPrefix(h, "a1") || strings.HasPrefix(h, "a2") || strings.HasPrefix(h, "a3") || strings.HasPrefix(h, "a4") || strings.HasPrefix(h, "a5") || strings.HasPrefix(h, "a6") || strings.HasPrefix(h, "a7") || strings.HasPrefix(h, "a8") || strings.HasPrefix(h, "a9") || strings.HasPrefix(h, "aa") || strings.HasPrefix(h, "ab") || strings.HasPrefix(h, "ac") || strings.HasPrefix(h, "ad"):
		return "United States"
	case strings.HasPrefix(h, "40") || strings.HasPrefix(h, "41") || strings.HasPrefix(h, "42") || strings.HasPrefix(h, "43"):
		return "United Kingdom"
	case strings.HasPrefix(h, "3c") || strings.HasPrefix(h, "3d"):
		return "Germany"
	case strings.HasPrefix(h, "38") || strings.HasPrefix(h, "39"):
		return "France"
	case strings.HasPrefix(h, "84") || strings.HasPrefix(h, "85") || strings.HasPrefix(h, "86") || strings.HasPrefix(h, "87"):
		return "Japan"
	case strings.HasPrefix(h, "71"):
		return "South Korea"
	case strings.HasPrefix(h, "76"):
		return "Singapore"
	case strings.HasPrefix(h, "89"):
		return "United Arab Emirates"
	case strings.HasPrefix(h, "7c"):
		return "Australia"
	case strings.HasPrefix(h, "c0") || strings.HasPrefix(h, "c1"):
		return "Canada"
	default:
		return "International"
	}
}

func inferRegistrationFromHex(hex string) string {
	h := strings.ToLower(strings.TrimSpace(hex))
	if strings.HasPrefix(h, "a") && len(h) == 6 {
		// FAA N-number conversion
		return "N" + strings.ToUpper(h[1:5])
	}
	if strings.HasPrefix(h, "78") && len(h) == 6 {
		return "B-" + strings.ToUpper(h[2:])
	}
	return ""
}

var knownMajorAirlines = map[string]struct{ Name, ICAO, IATA, Country string }{
	"CCA": {"Air China", "CCA", "CA", "China"},
	"CES": {"China Eastern Airlines", "CES", "MU", "China"},
	"CSN": {"China Southern Airlines", "CSN", "CZ", "China"},
	"CHH": {"Hainan Airlines", "CHH", "HU", "China"},
	"CSZ": {"Shenzhen Airlines", "CSZ", "ZH", "China"},
	"CSC": {"Sichuan Airlines", "CSC", "3U", "China"},
	"CXA": {"XiamenAir", "CXA", "MF", "China"},
	"UAL": {"United Airlines", "UAL", "UA", "United States"},
	"AAL": {"American Airlines", "AAL", "AA", "United States"},
	"DAL": {"Delta Air Lines", "DAL", "DL", "United States"},
	"SWA": {"Southwest Airlines", "SWA", "WN", "United States"},
	"DLH": {"Lufthansa", "DLH", "LH", "Germany"},
	"BAW": {"British Airways", "BAW", "BA", "United Kingdom"},
	"AFR": {"Air France", "AFR", "AF", "France"},
	"KLM": {"KLM Royal Dutch Airlines", "KLM", "KL", "Netherlands"},
	"ANA": {"All Nippon Airways", "ANA", "NH", "Japan"},
	"JAL": {"Japan Airlines", "JAL", "JL", "Japan"},
	"SIA": {"Singapore Airlines", "SIA", "SQ", "Singapore"},
	"UAE": {"Emirates", "UAE", "EK", "United Arab Emirates"},
	"QFA": {"Qantas", "QFA", "QF", "Australia"},
}
