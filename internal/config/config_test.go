package config

import (
	"os"
	"testing"
)

func TestOpenSkyCredentialsParsing(t *testing.T) {
	os.Setenv("CONTRAIL_SKIP_DOTENV", "true")
	defer os.Unsetenv("CONTRAIL_SKIP_DOTENV")

	// Clean env
	os.Unsetenv("OPENSKY_CLIENT_ID")
	os.Unsetenv("OPENSKY_CLIENT_SECRET")
	os.Unsetenv("OPENSKY_CREDENTIALS")
	os.Unsetenv("OPENSKY_TIER")
	os.Unsetenv("OPENSKY_DAILY_CREDIT_LIMIT")

	// Set JSON credentials
	os.Setenv("OPENSKY_CREDENTIALS", `{"clientId":"mock-client-id","clientSecret":"mock-client-secret-xyz"}`)
	defer os.Unsetenv("OPENSKY_CREDENTIALS")

	cfg := LoadConfig()

	if cfg.OpenSkyClientID != "mock-client-id" {
		t.Errorf("Expected OpenSkyClientID 'mock-client-id', got '%s'", cfg.OpenSkyClientID)
	}
	if cfg.OpenSkyClientSecret != "mock-client-secret-xyz" {
		t.Errorf("Expected OpenSkyClientSecret 'mock-client-secret-xyz', got '%s'", cfg.OpenSkyClientSecret)
	}
	if cfg.OpenSkyTier != "standard" {
		t.Errorf("Expected default tier 'standard', got '%s'", cfg.OpenSkyTier)
	}
	if cfg.OpenSkyDailyCreditLimit != 4000 {
		t.Errorf("Expected daily limit 4000 for standard tier, got %d", cfg.OpenSkyDailyCreditLimit)
	}
}

func TestOpenSkyTierLimits(t *testing.T) {
	os.Setenv("CONTRAIL_SKIP_DOTENV", "true")
	defer os.Unsetenv("CONTRAIL_SKIP_DOTENV")

	// Clean env
	os.Unsetenv("OPENSKY_CLIENT_ID")
	os.Unsetenv("OPENSKY_CLIENT_SECRET")
	os.Unsetenv("OPENSKY_CREDENTIALS")
	os.Unsetenv("OPENSKY_IS_FEEDER")
	os.Unsetenv("OPENSKY_DAILY_CREDIT_LIMIT")

	// Anonymous tier
	os.Setenv("OPENSKY_TIER", "anonymous")
	cfgAnon := LoadConfig()
	if cfgAnon.OpenSkyDailyCreditLimit != 400 {
		t.Errorf("Expected 400 daily limit for anonymous tier, got %d", cfgAnon.OpenSkyDailyCreditLimit)
	}

	// Feeder tier
	os.Setenv("OPENSKY_TIER", "feeder")
	cfgFeeder := LoadConfig()
	if cfgFeeder.OpenSkyDailyCreditLimit != 8000 {
		t.Errorf("Expected 8000 daily limit for feeder tier, got %d", cfgFeeder.OpenSkyDailyCreditLimit)
	}

	// Clean up
	os.Unsetenv("OPENSKY_TIER")
}

func init() {
	os.Setenv("JWT_SECRET", "test-jwt-secret-0123456789abcdef0123456789")
}
