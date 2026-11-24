package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// DB wraps the SQL database handle.
type DB struct {
	*sql.DB
	maxAgeResolver func(source string) float64
}

// SetMaxAgeResolver registers a dynamic function to compute dead reckoning max age per upstream source.
func (db *DB) SetMaxAgeResolver(resolver func(source string) float64) {
	db.maxAgeResolver = resolver
}

// InitDB initializes SQLite, ensures directory exists, applies PRAGMAs, and creates tables.
func InitDB(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	rawDB.SetMaxOpenConns(25)
	rawDB.SetMaxIdleConns(10)

	db := &DB{DB: rawDB}
	if err := db.migrate(); err != nil {
		rawDB.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS callsign_cache (
		callsign TEXT PRIMARY KEY,
		callsign_iata TEXT,
		airline_name TEXT,
		airline_icao TEXT,
		airline_iata TEXT,
		airline_country TEXT,
		origin_icao TEXT,
		origin_iata TEXT,
		origin_name TEXT,
		origin_city TEXT,
		origin_country TEXT,
		origin_lat REAL,
		origin_lon REAL,
		origin_elev INTEGER,
		dest_icao TEXT,
		dest_iata TEXT,
		dest_name TEXT,
		dest_city TEXT,
		dest_country TEXT,
		dest_lat REAL,
		dest_lon REAL,
		dest_elev INTEGER,
		route_summary TEXT,
		status TEXT, -- 'found', 'unknown'
		cached_at INTEGER,
		expires_at INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_callsign_expires ON callsign_cache(expires_at);

	CREATE TABLE IF NOT EXISTS aircraft_cache (
		hex TEXT PRIMARY KEY,
		registration TEXT,
		typecode TEXT,
		model TEXT,
		country TEXT,
		cached_at INTEGER,
		expires_at INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_aircraft_expires ON aircraft_cache(expires_at);

	CREATE TABLE IF NOT EXISTS live_flights (
		hex TEXT PRIMARY KEY,
		callsign_icao TEXT,
		callsign_iata TEXT,
		airline_name TEXT,
		airline_icao TEXT,
		airline_iata TEXT,
		origin_icao TEXT,
		origin_iata TEXT,
		origin_name TEXT,
		dest_icao TEXT,
		dest_iata TEXT,
		dest_name TEXT,
		route_summary TEXT,
		lat REAL,
		lon REAL,
		alt_baro_ft REAL,
		alt_geom_ft REAL,
		ground_speed_kts REAL,
		track_deg REAL,
		vertical_rate_fpm REAL,
		squawk TEXT,
		emergency_category TEXT,
		is_emergency INTEGER,
		is_diverting INTEGER,
		diversion_probability TEXT,
		candidate_alternate_icao TEXT,
		candidate_alternate_name TEXT,
		candidate_alternate_dist REAL,
		candidate_alternate_bearing REAL,
		raw_data_json TEXT,
		on_ground INTEGER,
		upstream_source TEXT,
		last_contact INTEGER,
		updated_at INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_live_callsign ON live_flights(callsign_icao);
	CREATE INDEX IF NOT EXISTS idx_live_emergency ON live_flights(is_emergency);
	CREATE INDEX IF NOT EXISTS idx_live_diverting ON live_flights(is_diverting);
	CREATE INDEX IF NOT EXISTS idx_live_origin ON live_flights(origin_icao);
	CREATE INDEX IF NOT EXISTS idx_live_dest ON live_flights(dest_icao);
	CREATE INDEX IF NOT EXISTS idx_live_updated ON live_flights(updated_at);
	CREATE INDEX IF NOT EXISTS idx_live_coords ON live_flights(lat, lon);

	CREATE TABLE IF NOT EXISTS flight_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hex TEXT,
		callsign_icao TEXT,
		callsign_iata TEXT,
		lat REAL,
		lon REAL,
		alt_baro_ft REAL,
		ground_speed_kts REAL,
		track_deg REAL,
		vertical_rate_fpm REAL,
		squawk TEXT,
		is_emergency INTEGER,
		is_diverting INTEGER,
		timestamp INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_snapshots_time ON flight_snapshots(timestamp);
	CREATE INDEX IF NOT EXISTS idx_snapshots_hex_time ON flight_snapshots(hex, timestamp);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT NOT NULL,
		purpose TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		status TEXT NOT NULL DEFAULT 'active',
		terms_agreed_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	CREATE TABLE IF NOT EXISTS api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		key_name TEXT NOT NULL,
		key_prefix TEXT NOT NULL,
		key_hash TEXT UNIQUE NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		terms_agreed INTEGER NOT NULL DEFAULT 1,
		created_at INTEGER NOT NULL,
		last_used_at INTEGER,
		total_requests INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_keys_hash ON api_keys(key_hash);
	CREATE INDEX IF NOT EXISTS idx_keys_user ON api_keys(user_id);

	CREATE TABLE IF NOT EXISTS api_key_usage_daily (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
		date_str TEXT NOT NULL,
		request_count INTEGER DEFAULT 0,
		UNIQUE(key_id, date_str)
	);

	CREATE TABLE IF NOT EXISTS email_verifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL,
		code TEXT NOT NULL,
		purpose TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		used INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_verif_email ON email_verifications(email, purpose);

	CREATE TABLE IF NOT EXISTS rate_limit_alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		key_id INTEGER NOT NULL,
		reason TEXT NOT NULL,
		sent_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS upstream_fetch_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		provider TEXT NOT NULL,
		hub TEXT DEFAULT '',
		status TEXT NOT NULL,
		status_code INTEGER NOT NULL,
		aircraft_count INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		error_message TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_upstream_logs_ts ON upstream_fetch_logs(timestamp DESC);

	CREATE TABLE IF NOT EXISTS upstream_state (
		provider TEXT PRIMARY KEY,
		status TEXT NOT NULL DEFAULT 'ok',
		last_success_at INTEGER NOT NULL DEFAULT 0,
		next_allowed_at INTEGER NOT NULL DEFAULT 0,
		remaining_credits INTEGER NOT NULL DEFAULT 0,
		consecutive_failures INTEGER NOT NULL DEFAULT 0,
		cooldown_until INTEGER NOT NULL DEFAULT 0,
		last_fetched_count INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS opensky_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		query_type TEXT NOT NULL DEFAULT 'global',
		credits_used INTEGER NOT NULL,
		status_code INTEGER NOT NULL,
		rate_limit_remaining INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_opensky_usage_ts ON opensky_usage(timestamp);

	CREATE TABLE IF NOT EXISTS adsblol_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		endpoint TEXT NOT NULL,
		status_code INTEGER NOT NULL,
		count INTEGER NOT NULL DEFAULT 1
	);
	CREATE INDEX IF NOT EXISTS idx_adsblol_usage_ts ON adsblol_usage(timestamp);

	CREATE TABLE IF NOT EXISTS extrapolation_accuracy_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hex TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		extrapolated_lat REAL NOT NULL,
		extrapolated_lon REAL NOT NULL,
		observed_lat REAL NOT NULL,
		observed_lon REAL NOT NULL,
		error_km REAL NOT NULL,
		age_seconds INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_extrap_acc_ts ON extrapolation_accuracy_logs(timestamp);

	CREATE TABLE IF NOT EXISTS route_corridors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		origin_icao TEXT NOT NULL,
		dest_icao TEXT NOT NULL,
		sample_count INTEGER NOT NULL,
		corridor_data_json TEXT NOT NULL,
		updated_at INTEGER NOT NULL,
		UNIQUE(origin_icao, dest_icao)
	);
	CREATE INDEX IF NOT EXISTS idx_corridors_pair ON route_corridors(origin_icao, dest_icao);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Safe idempotent column additions
	db.runMigrations()

	// Seed Admin user if absent
	return db.seedDefaults()
}

func (db *DB) runMigrations() {
	// Add hub column to upstream_fetch_logs if missing
	_, _ = db.Exec("ALTER TABLE upstream_fetch_logs ADD COLUMN hub TEXT DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE upstream_fetch_logs ADD COLUMN repeat_count INTEGER DEFAULT 1")
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_upstream_logs_prov_hub ON upstream_fetch_logs(provider, hub)")
	// Add kinematics tracking columns to live_flights if missing
	_, _ = db.Exec("ALTER TABLE live_flights ADD COLUMN position_source TEXT DEFAULT 'observed'")
	_, _ = db.Exec("ALTER TABLE live_flights ADD COLUMN estimated_age_s INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE live_flights ADD COLUMN last_observed_at INTEGER DEFAULT 0")
	// Add query_type to opensky_usage if missing
	_, _ = db.Exec("ALTER TABLE opensky_usage ADD COLUMN query_type TEXT NOT NULL DEFAULT 'global'")
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_opensky_usage_type ON opensky_usage(query_type, timestamp)")
}

func (db *DB) seedDefaults() error {
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminEmail == "" || adminPass == "" {
		return nil // no admin seeded unless ADMIN_EMAIL and ADMIN_PASSWORD are set
	}

	var count int
	_ = db.QueryRow("SELECT count(*) FROM users WHERE email = ?", adminEmail).Scan(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err == nil {
			now := time.Now().Unix()
			res, err := db.Exec(`
				INSERT INTO users (email, password_hash, name, purpose, role, status, terms_agreed_at, created_at, updated_at)
				VALUES (?, ?, 'Jake (Administrator)', 'System Management and API Governance', 'admin', 'active', ?, ?, ?)
			`, adminEmail, string(hash), now, now, now)
			if err == nil {
				adminUserID, _ := res.LastInsertId()
				// Provision default admin key
				_, _, _ = db.GenerateAPIKey(adminUserID, "Jake Admin Master Key")
			}
		}
	}

	// Seed default field restrictions if not set
	var settingCount int
	_ = db.QueryRow("SELECT count(*) FROM system_settings WHERE key = 'normal_user_allowed_fields'").Scan(&settingCount)
	if settingCount == 0 {
		defaultFields := `["flight","aircraft","telemetry","route","emergency","diversion_analysis","metadata"]`
		_, _ = db.Exec("INSERT INTO system_settings (key, value) VALUES ('normal_user_allowed_fields', ?)", defaultFields)
	}

	var regSettingCount int
	_ = db.QueryRow("SELECT count(*) FROM system_settings WHERE key = 'registration_enabled'").Scan(&regSettingCount)
	if regSettingCount == 0 {
		_, _ = db.Exec("INSERT INTO system_settings (key, value) VALUES ('registration_enabled', 'true')")
	}

	return nil
}
