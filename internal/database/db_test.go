package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"contrail/internal/models"
)

func strPtr(s string) *string {
	return &s
}

func TestDatabaseAndCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contrail-db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Test 1: Callsign cache set and get
	airline := &models.AirlineInfo{
		Name:    "American Airlines",
		ICAO:    "AAL",
		IATA:    "AA",
		Country: "United States",
	}
	route := &models.RouteInfo{
		Summary: "JFK -> LAX",
		Origin: &models.AirportInfo{
			ICAO: "KJFK",
			IATA: "JFK",
			Name: "John F Kennedy",
			Location: models.AirportLocation{
				Latitude:  40.6398,
				Longitude: -73.7789,
			},
		},
		Destination: &models.AirportInfo{
			ICAO: "KLAX",
			IATA: "LAX",
			Name: "Los Angeles",
			Location: models.AirportLocation{
				Latitude:  33.9425,
				Longitude: -118.408,
			},
		},
	}

	err = db.SetCachedCallsign("AAL1", "AA1", airline, route, "found", 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set cached callsign: %v", err)
	}

	cached, found, err := db.GetCachedCallsign("AAL1")
	if err != nil || !found {
		t.Fatalf("Failed to get cached callsign: found=%v, err=%v", found, err)
	}
	if cached.IATA != "AA1" || cached.Route.Origin.ICAO != "KJFK" {
		t.Errorf("Unexpected cached route data: %+v", cached)
	}

	// Test 2: SaveLiveFlights and GetLiveFlights
	testFlight := models.FlightData{
		Aircraft: models.AircraftInfo{
			Hex: "a00001",
		},
		Flight: models.FlightIdentifier{
			CallsignICAO: "AAL1",
			CallsignIATA: strPtr("AA1"),
			Airline:      airline,
		},
		Route: *route,
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  38.5,
				Longitude: -95.0,
			},
			Altitude: models.AltitudeInfo{
				BarometricFt: 35000,
			},
			Speed: models.SpeedInfo{
				GroundSpeedKts: 460,
			},
			Attitude: models.AttitudeInfo{
				TrackDeg: 260,
			},
			OnGround: false,
		},
		Emergency: models.EmergencyInfo{
			Squawk:      "1200",
			IsEmergency: false,
			Category:    models.EmergencyNone,
		},
		DiversionAnalysis: models.DiversionAnalysis{
			PredictedDiversion: false,
			ProbabilityLevel:   "NONE",
		},
		Metadata: models.MetadataInfo{
			UpstreamSource: "adsb_fi",
			LastUpdated:    time.Now(),
		},
	}

	err = db.SaveLiveFlights([]models.FlightData{testFlight})
	if err != nil {
		t.Fatalf("Failed to save live flights: %v", err)
	}

	flights, count, err := db.GetLiveFlights(models.FlightQueryFilter{CallsignPrefix: "AAL"}, "last_seen", "desc", 10, 0)
	if err != nil {
		t.Fatalf("Failed to query live flights: %v", err)
	}
	if count != 1 || len(flights) != 1 {
		t.Errorf("Expected 1 flight, got %d (matching: %d)", len(flights), count)
	}

	// Test 3: GetFlightByIdentifier
	single, err := db.GetFlightByIdentifier("a00001")
	if err != nil || single == nil {
		t.Fatalf("Failed to retrieve single flight by hex: %v", err)
	}
	if single.Flight.CallsignICAO != "AAL1" {
		t.Errorf("Expected callsign AAL1, got %s", single.Flight.CallsignICAO)
	}

	// Test 4: Retention cleanup
	pruned, err := db.CleanOldSnapshots(90)
	if err != nil {
		t.Fatalf("Failed running retention cleanup: %v", err)
	}
	if pruned != 0 {
		t.Errorf("Expected 0 pruned snapshots for fresh DB, got %d", pruned)
	}
}
