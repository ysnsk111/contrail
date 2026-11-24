package kinematics

import (
	"math"
	"testing"
	"time"

	"contrail/internal/models"
)

func TestExtrapolate(t *testing.T) {
	past := time.Unix(1700000000, 0).UTC()
	now := time.Unix(1700000060, 0).UTC() // exactly 60 seconds later

	fd := &models.FlightData{
		Aircraft: models.AircraftInfo{
			Hex: "484557",
		},
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  30.0,
				Longitude: 120.0,
			},
			Speed: models.SpeedInfo{
				GroundSpeedKts: 480.0,
			},
			Attitude: models.AttitudeInfo{
				TrackDeg:        90.0, // Flying directly east
				VerticalRateFpm: 1000.0,
			},
			Altitude: models.AltitudeInfo{
				BarometricFt: 30000.0,
			},
			LastContactEpoch: past.Unix(),
		},
	}

	Extrapolate(fd, now, 180.0)

	// Longitude should have increased (flying east)
	if fd.Telemetry.Position.Longitude <= 120.0 {
		t.Errorf("expected longitude to increase from 120.0, got %f", fd.Telemetry.Position.Longitude)
	}

	// Latitude should stay very close to 30.0
	if math.Abs(fd.Telemetry.Position.Latitude-30.0) > 0.05 {
		t.Errorf("expected latitude close to 30.0, got %f", fd.Telemetry.Position.Latitude)
	}

	// Position source should be estimated
	if fd.PositionSource != "estimated" || fd.Telemetry.PositionSource != "estimated" {
		t.Errorf("expected position_source 'estimated', got %s", fd.PositionSource)
	}
	if fd.EstimatedAgeS != 60 {
		t.Errorf("expected estimated_age_s 60, got %d", fd.EstimatedAgeS)
	}

	// Altitude should have climbed with decay
	if fd.Telemetry.Altitude.BarometricFt <= 30000.0 {
		t.Errorf("expected altitude to increase above 30000, got %f", fd.Telemetry.Altitude.BarometricFt)
	}
}

func TestExtrapolateGroundStationary(t *testing.T) {
	past := time.Unix(1700000000, 0).UTC()
	now := time.Unix(1700000060, 0).UTC()

	fd := &models.FlightData{
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  30.0,
				Longitude: 120.0,
			},
			OnGround:         true,
			Speed:            models.SpeedInfo{GroundSpeedKts: 0},
			LastContactEpoch: past.Unix(),
		},
	}

	Extrapolate(fd, now, 180.0)
	if fd.Telemetry.Position.Latitude != 30.0 || fd.Telemetry.Position.Longitude != 120.0 {
		t.Errorf("aircraft on ground should not extrapolate position")
	}
	if fd.PositionSource != "observed" {
		t.Errorf("expected position_source 'observed' for on-ground aircraft, got %s", fd.PositionSource)
	}
}

func TestExtrapolateLowSpeed(t *testing.T) {
	past := time.Unix(1700000000, 0).UTC()
	now := time.Unix(1700000060, 0).UTC()

	fd := &models.FlightData{
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  30.0,
				Longitude: 120.0,
			},
			Speed:            models.SpeedInfo{GroundSpeedKts: 45.0}, // < 50 kt
			LastContactEpoch: past.Unix(),
		},
	}

	Extrapolate(fd, now, 180.0)
	if fd.Telemetry.Position.Longitude != 120.0 {
		t.Errorf("aircraft speed < 50kt must not extrapolate position")
	}
	if fd.PositionSource != "observed" {
		t.Errorf("expected position_source 'observed', got %s", fd.PositionSource)
	}
}

func TestExtrapolateTerminalPhase(t *testing.T) {
	past := time.Unix(1700000000, 0).UTC()
	now := time.Unix(1700000060, 0).UTC()

	// Aircraft at 5,000 ft within 10 km of destination airport
	fd := &models.FlightData{
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  31.14,
				Longitude: 121.80,
			},
			Speed:            models.SpeedInfo{GroundSpeedKts: 180.0},
			Altitude:         models.AltitudeInfo{BarometricFt: 5000.0},
			LastContactEpoch: past.Unix(),
		},
		Route: models.RouteInfo{
			Destination: &models.AirportInfo{
				ICAO: "ZSPD",
				Location: models.AirportLocation{
					Latitude:  31.143,
					Longitude: 121.805, // ~0.5 km away
				},
			},
		},
	}

	Extrapolate(fd, now, 180.0)
	// Should NOT extrapolate because below 10,000 ft and within 30 km of airport
	if fd.PositionSource != "observed" {
		t.Errorf("expected position_source 'observed' for terminal arrival phase, got %s", fd.PositionSource)
	}
	if fd.Telemetry.Position.Latitude != 31.14 {
		t.Errorf("terminal arrival aircraft coordinates should not advance")
	}
}

func TestExtrapolateAltitudeCeilingClamped(t *testing.T) {
	past := time.Unix(1700000000, 0).UTC()
	now := time.Unix(1700000060, 0).UTC()

	fd := &models.FlightData{
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  30.0,
				Longitude: 120.0,
			},
			Speed: models.SpeedInfo{GroundSpeedKts: 450.0},
			Altitude: models.AltitudeInfo{
				BarometricFt: 44800.0,
			},
			Attitude: models.AttitudeInfo{
				TrackDeg:        45.0,
				VerticalRateFpm: 3000.0, // Would exceed 45000
			},
			LastContactEpoch: past.Unix(),
		},
	}

	Extrapolate(fd, now, 180.0)
	if fd.Telemetry.Altitude.BarometricFt > 45000.0 {
		t.Errorf("extrapolated altitude must be clamped to 45000 ft, got %f", fd.Telemetry.Altitude.BarometricFt)
	}
}

func TestExtrapolateMaxAgeClamping(t *testing.T) {
	past := time.Unix(1700000000, 0).UTC()
	// 300 seconds elapsed (5 minutes), but maxAge is 180s (3 minutes)
	now := time.Unix(1700000300, 0).UTC()

	fd := &models.FlightData{
		Telemetry: models.TelemetryInfo{
			Position: models.PositionInfo{
				Latitude:  30.0,
				Longitude: 120.0,
			},
			Speed: models.SpeedInfo{GroundSpeedKts: 480.0},
			Attitude: models.AttitudeInfo{
				TrackDeg: 90.0,
			},
			Altitude: models.AltitudeInfo{
				BarometricFt: 30000.0,
			},
			LastContactEpoch: past.Unix(),
		},
	}

	Extrapolate(fd, now, 180.0)

	// Position should match 180 seconds of travel, NOT 300 seconds!
	// 180s at 480 kts = 24 nm ~ 44.4 km
	// 300s at 480 kts = 40 nm ~ 74.0 km
	distMoved := HaversineDistance(30.0, 120.0, fd.Telemetry.Position.Latitude, fd.Telemetry.Position.Longitude)
	if math.Abs(distMoved-44.4) > 3.0 {
		t.Errorf("expected distance moved around 44.4 km (clamped to 180s), got %f km", distMoved)
	}
	if fd.PositionSource != "stale" {
		t.Errorf("expected position_source 'stale', got %s", fd.PositionSource)
	}
	if fd.EstimatedAgeS != 300 {
		t.Errorf("expected estimated_age_s to reflect true elapsed 300s, got %d", fd.EstimatedAgeS)
	}
}

func TestCalculateBearing(t *testing.T) {
	// London (51.5, 0.0) to Paris (48.8, 2.3) ~ South-East (~150 deg)
	bearing := CalculateBearing(51.5, 0.0, 48.8, 2.3)
	if bearing < 140 || bearing > 170 {
		t.Errorf("expected bearing between 140 and 170, got %f", bearing)
	}
}
