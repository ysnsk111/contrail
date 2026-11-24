package config

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application runtime configuration.
type Config struct {
	// Server
	Port       string
	APIBaseURL string

	// Database
	DBPath string

	// Poller Settings
	PollerEnabled           bool
	UpstreamProvider        string // "auto", "adsb_fi", "opensky", "airplanes_live"
	ADSBFiSnapshotURL       string
	ADSBFiPollInterval      time.Duration
	OpenSkyURL              string
	OpenSkyClientID         string
	OpenSkyClientSecret     string
	OpenSkyIsFeeder         bool
	OpenSkyTier             string
	OpenSkyPollInterval     time.Duration
	OpenSkyBoundingBox      string // lamin,lomin,lamax,lomax (optional)
	OpenSkyMinIntervalSec          int
	OpenSkyCostPerCall             int
	OpenSkyDailyCreditLimit        int
	OpenSkyGlobalShare             float64
	ADSBLolHubPolling              bool
	ADSBLolDailyLimit              int
	ADSBLolAPIKey                  string
	ADSBLolMinIntervalSec          int
	ADSBLolEmergencyIntervalSec    int
	ADSBLolCalibrationCooldownSec  int
	ADSBMeshRateLimitRPS           float64
	ADSBMeshRecoveryIntervalSec    float64
	ADSBMeshGlobalCooldownMin      int
	DeadReckoningGlobalCycles      float64
	DeadReckoningCruiseMaxAgeSec   int
	DeadReckoningMaxAgeSec         int
	AircraftPurgeTimeoutSec        int
	WeatherWindsEnabled            bool
	WeatherRadarLayerEnabled       bool
	AlertNotifyEmail               string
	SourcePriority                 []string

	// Route Enricher (adsbdb)
	ADSBDbBaseURL           string
	ADSBDbRateLimitPerMin   int
	CallsignCacheTTLHours   int

	// Data Retention (3 months default = 90 days)
	RetentionDays           int
	RetentionCleanupInterval time.Duration

	// Mailer & SMTP
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// Auth & Security
	JWTSecret       string
	RateLimitRPM    int
	RateLimitRPH    int
	RateLimitDaily  int
	AdminEmail      string
	AdminPassword   string

	// Cap (anti-bot CAPTCHA)
	CapSiteKey      string
	CapSecretKey    string
	CapVerifyURL    string
	CapAPIEndpoint  string
}

// LoadConfig loads configuration from environment variables with safe defaults.
func LoadConfig() *Config {
	if os.Getenv("CONTRAIL_SKIP_DOTENV") != "true" {
		loadEnvFile(".env")
		loadEnvFile("/home/suki/project/Contrail/.env")
	}

	port := getEnv("PORT", "8080")
	if !strings.HasPrefix(port, ":") && !strings.Contains(port, ":") {
		// Just numeric port
	}

	apiBaseURL := getEnv("API_BASE_URL", "https://airline.api.veyino.org")
	dbPath := getEnv("DB_PATH", "./data/contrail.db")

	pollerEnabled := getEnvBool("POLLER_ENABLED", true)
	upstreamProvider := strings.ToLower(getEnv("UPSTREAM_PROVIDER", "auto"))

	adsbFiSnapshotURL := getEnv("ADSB_FI_SNAPSHOT_URL", "https://opendata.adsb.fi/api/v2/snapshot")
	adsbFiIntervalSec := getEnvInt("ADSB_FI_POLL_INTERVAL_SEC", 35) // 35s (> 30s limit for feeder)

	openSkyURL := getEnv("OPENSKY_URL", "https://opensky-network.org/api/states/all")
	openSkyClientID := getEnv("OPENSKY_CLIENT_ID", "")
	openSkyClientSecret := getEnv("OPENSKY_CLIENT_SECRET", "")
	openSkyCredentialsRaw := getEnv("OPENSKY_CREDENTIALS", "")
	if openSkyCredentialsRaw != "" {
		var creds struct {
			ClientID     string `json:"clientId"`
			ClientSecret string `json:"clientSecret"`
			ClientIDAlt  string `json:"client_id"`
			ClientSecAlt string `json:"client_secret"`
		}
		if err := json.Unmarshal([]byte(openSkyCredentialsRaw), &creds); err == nil {
			if openSkyClientID == "" {
				if creds.ClientID != "" {
					openSkyClientID = creds.ClientID
				} else if creds.ClientIDAlt != "" {
					openSkyClientID = creds.ClientIDAlt
				}
			}
			if openSkyClientSecret == "" {
				if creds.ClientSecret != "" {
					openSkyClientSecret = creds.ClientSecret
				} else if creds.ClientSecAlt != "" {
					openSkyClientSecret = creds.ClientSecAlt
				}
			}
		}
	}

	openSkyIsFeeder := getEnvBool("OPENSKY_IS_FEEDER", false)
	openSkyBBox := getEnv("OPENSKY_BOUNDING_BOX", "")

	openSkyTier := strings.ToLower(getEnv("OPENSKY_TIER", ""))
	if openSkyTier == "" {
		if openSkyIsFeeder {
			openSkyTier = "feeder"
		} else if openSkyClientID != "" {
			openSkyTier = "standard"
		} else {
			openSkyTier = "anonymous"
		}
	}

	// OpenSky quotas by tier:
	// - anonymous: 400 credits/day (IP bucket)
	// - standard: 4,000 credits/day
	// - feeder: 8,000 credits/day
	// - authorized: 14,400 credits/hour -> 345,600 credits/day
	defaultDailyLimit := 4000
	defaultOpenSkyInterval := 95
	switch openSkyTier {
	case "feeder":
		defaultDailyLimit = 8000
		defaultOpenSkyInterval = 50
	case "authorized":
		defaultDailyLimit = 345600
		defaultOpenSkyInterval = 10
	case "anonymous":
		defaultDailyLimit = 400
		defaultOpenSkyInterval = 900 // 400 credits / 4 = 100 calls/day -> ~864s
	case "standard":
		fallthrough
	default:
		defaultDailyLimit = 4000
		defaultOpenSkyInterval = 95
	}

	openSkyIntervalSec := getEnvInt("OPENSKY_POLL_INTERVAL_SEC", defaultOpenSkyInterval)
	openSkyMinIntervalSec := getEnvInt("OPENSKY_MIN_INTERVAL_SEC", 100)
	openSkyCostPerCall := getEnvInt("OPENSKY_COST_PER_CALL", 4)
	openSkyDailyCreditLimit := getEnvInt("OPENSKY_DAILY_CREDIT_LIMIT", defaultDailyLimit)
	openSkyGlobalShare := getEnvFloat("OPENSKY_GLOBAL_SHARE", 0.65)
	adsbLolHubPolling := getEnvBool("ADSBLOL_HUB_POLLING", false)
	adsbLolDailyLimit := getEnvInt("ADSBLOL_DAILY_LIMIT", 450)
	adsbLolAPIKey := getEnv("ADSBLOL_API_KEY", "")
	adsbLolMinIntervalSec := getEnvInt("ADSBLOL_MIN_INTERVAL_SEC", 30)
	adsbLolEmergencyIntervalSec := getEnvInt("ADSBLOL_EMERGENCY_INTERVAL_SEC", 600)
	adsbLolCalibrationCooldownSec := getEnvInt("ADSBLOL_CALIBRATION_COOLDOWN_SEC", 600)
	adsbMeshRateLimitRPS := getEnvFloat("ADSB_MESH_RATE_LIMIT_RPS", 1.0)
	adsbMeshRecoveryIntervalSec := getEnvFloat("ADSB_MESH_RECOVERY_INTERVAL_SEC", 3.5) // 3-5 sec recovery interval
	adsbMeshGlobalCooldownMin := getEnvInt("ADSB_MESH_GLOBAL_COOLDOWN_MIN", 15)        // 15 min default
	deadReckoningGlobalCycles := getEnvFloat("DEAD_RECKONING_GLOBAL_CYCLES", 2.5)      // 2.5 global cycles (~5.5 min)
	deadReckoningCruiseMaxAgeSec := getEnvInt("DEAD_RECKONING_CRUISE_MAX_AGE_SEC", 720) // 12 min cruise max
	deadReckoningMaxAgeSec := getEnvInt("DEAD_RECKONING_MAX_AGE_SEC", 330)             // 5.5 min base default
	aircraftPurgeTimeoutSec := getEnvInt("AIRCRAFT_PURGE_TIMEOUT_SEC", 1800)           // 30 min default
	weatherWindsEnabled := getEnvBool("WEATHER_WINDS_ENABLED", false)
	weatherRadarLayerEnabled := getEnvBool("WEATHER_RADAR_LAYER_ENABLED", false)
	alertNotifyEmail := getEnv("ALERT_NOTIFY_EMAIL", "")
	sourcePriorityRaw := getEnv("SOURCE_PRIORITY", "opensky,adsb_mesh")
	sourcePriority := strings.Split(sourcePriorityRaw, ",")
	for i := range sourcePriority {
		sourcePriority[i] = strings.ToLower(strings.TrimSpace(sourcePriority[i]))
	}

	adsbDbBaseURL := getEnv("ADS_B_DB_BASE_URL", "https://api.adsbdb.com/v0/callsign/")
	adsbDbRateLimitPerMin := getEnvInt("ADS_B_DB_RATE_LIMIT_PER_MIN", 400) // 400 req/min (< 512 limit)
	callsignCacheTTLHours := getEnvInt("CALLSIGN_CACHE_TTL_HOURS", 24)

	retentionDays := getEnvInt("RETENTION_DAYS", 90) // 3 months
	retentionCleanupIntervalMin := getEnvInt("RETENTION_CLEANUP_INTERVAL_MIN", 60)

	smtpHost := getEnv("SMTP_HOST", "127.0.0.1")
	smtpPort := getEnvInt("SMTP_PORT", 587)
	smtpUser := getEnv("SMTP_USER", "")
	smtpPass := getEnv("SMTP_PASS", "")
	smtpFrom := getEnv("SMTP_FROM", "Contrail <no-reply@veyino.org>")

	jwtSecret := getEnv("JWT_SECRET", "")
	rateLimitRPM := getEnvInt("RATE_LIMIT_RPM", 60)
	rateLimitRPH := getEnvInt("RATE_LIMIT_RPH", 1000)
	rateLimitDaily := getEnvInt("RATE_LIMIT_DAILY", 10000)

	adminEmail := getEnv("ADMIN_EMAIL", "")
	adminPassword := getEnv("ADMIN_PASSWORD", "")

	capSiteKey := getEnv("CAP_SITE_KEY", "b12b141f33")
	capSecretKey := getEnv("CAP_SECRET_KEY", "")
	capVerifyURL := getEnv("CAP_VERIFY_URL", "http://127.0.0.1:13000/b12b141f33/siteverify")
	capAPIEndpoint := getEnv("CAP_API_ENDPOINT", "/cap/b12b141f33/")

	return &Config{
		Port:                     port,
		APIBaseURL:               apiBaseURL,
		DBPath:                   dbPath,
		PollerEnabled:            pollerEnabled,
		UpstreamProvider:         upstreamProvider,
		ADSBFiSnapshotURL:        adsbFiSnapshotURL,
		ADSBFiPollInterval:       time.Duration(adsbFiIntervalSec) * time.Second,
		OpenSkyURL:               openSkyURL,
		OpenSkyClientID:          openSkyClientID,
		OpenSkyClientSecret:      openSkyClientSecret,
		OpenSkyIsFeeder:          openSkyIsFeeder,
		OpenSkyTier:              openSkyTier,
		OpenSkyPollInterval:      time.Duration(openSkyIntervalSec) * time.Second,
		OpenSkyBoundingBox:       openSkyBBox,
		OpenSkyMinIntervalSec:       openSkyMinIntervalSec,
		OpenSkyCostPerCall:          openSkyCostPerCall,
		OpenSkyDailyCreditLimit:     openSkyDailyCreditLimit,
		OpenSkyGlobalShare:          openSkyGlobalShare,
		ADSBLolHubPolling:           adsbLolHubPolling,
		ADSBLolDailyLimit:           adsbLolDailyLimit,
		ADSBLolAPIKey:               adsbLolAPIKey,
		ADSBLolMinIntervalSec:       adsbLolMinIntervalSec,
		ADSBLolEmergencyIntervalSec: adsbLolEmergencyIntervalSec,
		ADSBLolCalibrationCooldownSec: adsbLolCalibrationCooldownSec,
		ADSBMeshRateLimitRPS:        adsbMeshRateLimitRPS,
		ADSBMeshRecoveryIntervalSec: adsbMeshRecoveryIntervalSec,
		ADSBMeshGlobalCooldownMin:   adsbMeshGlobalCooldownMin,
		DeadReckoningGlobalCycles:   deadReckoningGlobalCycles,
		DeadReckoningCruiseMaxAgeSec: deadReckoningCruiseMaxAgeSec,
		DeadReckoningMaxAgeSec:      deadReckoningMaxAgeSec,
		AircraftPurgeTimeoutSec:     aircraftPurgeTimeoutSec,
		WeatherWindsEnabled:         weatherWindsEnabled,
		WeatherRadarLayerEnabled:    weatherRadarLayerEnabled,
		AlertNotifyEmail:            alertNotifyEmail,
		SourcePriority:              sourcePriority,
		ADSBDbBaseURL:            adsbDbBaseURL,
		ADSBDbRateLimitPerMin:    adsbDbRateLimitPerMin,
		CallsignCacheTTLHours:    callsignCacheTTLHours,
		RetentionDays:            retentionDays,
		RetentionCleanupInterval: time.Duration(retentionCleanupIntervalMin) * time.Minute,
		SMTPHost:                 smtpHost,
		SMTPPort:                 smtpPort,
		SMTPUser:                 smtpUser,
		SMTPPass:                 smtpPass,
		SMTPFrom:                 smtpFrom,
		JWTSecret:                jwtSecret,
		RateLimitRPM:             rateLimitRPM,
		RateLimitRPH:             rateLimitRPH,
		RateLimitDaily:           rateLimitDaily,
		AdminEmail:               adminEmail,
		AdminPassword:            adminPassword,
		CapSiteKey:               capSiteKey,
		CapSecretKey:             capSecretKey,
		CapVerifyURL:             capVerifyURL,
		CapAPIEndpoint:           capAPIEndpoint,
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		lower := strings.ToLower(strings.TrimSpace(val))
		if lower == "true" || lower == "1" || lower == "yes" {
			return true
		}
		if lower == "false" || lower == "0" || lower == "no" {
			return false
		}
	}
	return defaultVal
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
