package poller

import (
	"testing"
	"time"

	"contrail/internal/models"
)

func TestDiversionPrediction(t *testing.T) {
	airportDB := GetAirportDB()
	engine := NewDiversionEngine(airportDB)

	// Scheduled destination: Beijing Capital (ZBAA)
	destApt := airportDB.Lookup("ZBAA")
	if destApt == nil {
		t.Fatalf("Failed to find ZBAA airport in AirportDB")
	}

	route := &models.RouteInfo{
		Summary: "SHA -> PEK",
		Origin: &models.AirportInfo{
			ICAO: "ZSSS",
			Name: "Shanghai Hongqiao",
		},
		Destination: destApt,
	}

	// Normal en-route flight at FL350 heading towards destination
	destLat := destApt.Location.Latitude
	destLon := destApt.Location.Longitude
	bearingToDest := InitialBearing(36.0, 117.0, destLat, destLon)

	normalAC := &models.RawAircraftSnapshot{
		Hex:          "a1b2c3",
		Flight:       "CES5101",
		Squawk:       "1200",
		Lat:          36.0,
		Lon:          117.0,
		AltBaroFt:    35000,
		GroundSpeed:  450,
		Track:        bearingToDest, // Aligned with destination
		VerticalRate: 0,
		OnGround:     false,
	}

	analysisNormal := engine.Analyze(normalAC, route)
	if analysisNormal.PredictedDiversion {
		t.Errorf("Expected normal flight to not be predicting diversion, got true")
	}

	// Scenario 1: Emergency squawk 7700
	emergAC := &models.RawAircraftSnapshot{
		Hex:          "e1e1e1",
		Flight:       "CES9999",
		Squawk:       "7700",
		Lat:          36.0,
		Lon:          117.0,
		AltBaroFt:    8500, // < 10,000ft
		GroundSpeed:  280,
		Track:        mathMod(bearingToDest+90, 360), // Deviating by 90°
		VerticalRate: -1800,                          // Rapid descent
		OnGround:     false,
	}

	// First pass
	_ = engine.Analyze(emergAC, route)
	// Second pass after small time to establish sustained deviation
	time.Sleep(50 * time.Millisecond)
	analysisEmerg := engine.Analyze(emergAC, route)

	if !analysisEmerg.PredictedDiversion {
		t.Errorf("Expected emergency descending flight to predict diversion, got false")
	}
	if analysisEmerg.ProbabilityLevel != "CRITICAL" {
		t.Errorf("Expected CRITICAL probability for Squawk 7700, got %s", analysisEmerg.ProbabilityLevel)
	}

	if len(analysisEmerg.TriggeredCriteria) < 2 {
		t.Errorf("Expected at least 2 triggered criteria, got %d", len(analysisEmerg.TriggeredCriteria))
	}

	// Scenario 2: Same origin and destination (survey/circular flight)
	circularRoute := &models.RouteInfo{
		Summary: "PEK -> PEK",
		Origin: &models.AirportInfo{
			ICAO: "ZBAA",
			Name: "Beijing Capital",
		},
		Destination: destApt,
	}
	analysisCircular := engine.Analyze(emergAC, circularRoute)
	if analysisCircular.PredictedDiversion {
		t.Errorf("Expected circular flight to not predict diversion, got true")
	}

	// Scenario 3: Landing at destination (Alt < 3,000ft near destination)
	landingAC := &models.RawAircraftSnapshot{
		Hex:          "b2c3d4",
		Flight:       "CES1234",
		Squawk:       "1200",
		Lat:          destLat + 0.05, // ~6km away
		Lon:          destLon + 0.05,
		AltBaroFt:    1800,
		GroundSpeed:  140,
		Track:        bearingToDest,
		VerticalRate: -700,
		OnGround:     false,
	}
	analysisLanding := engine.Analyze(landingAC, route)
	if analysisLanding.PredictedDiversion {
		t.Errorf("Expected normal landing flight near destination to not predict diversion, got true")
	}
}

func mathMod(a, b float64) float64 {
	for a < 0 {
		a += b
	}
	for a >= b {
		a -= b
	}
	return a
}
