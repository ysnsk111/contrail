package kinematics

import (
	"math"
	"time"

	"contrail/internal/models"
)

const (
	EarthRadiusKm       = 6371.0
	KnotsToKmh          = 1.852
	FreshObservationSec = 30.0 // Points younger than 30s are strictly "observed"
)

// WindCorrectionHook provides optional high-altitude wind corrections (e.g. NOAA GFS vector field).
var WindCorrectionHook func(lat, lon, altFt, speedKts, trackDeg float64) (deltaSpeedKts float64, deltaTrackDeg float64)

// Extrapolate advances an aircraft's position and altitude based on its ground speed,
// heading track, vertical rate, and elapsed time since last sensor contact.
// It wraps ExtrapolateWithCruise with maxAgeSec applied to both base and cruise horizons.
func Extrapolate(fd *models.FlightData, now time.Time, maxAgeSec float64) {
	if maxAgeSec <= 0 {
		ExtrapolateWithCruise(fd, now, 330.0, 720.0)
	} else {
		ExtrapolateWithCruise(fd, now, maxAgeSec, maxAgeSec)
	}
}

// ClassifyFlightStatus provides a unified status determination ("observed", "estimated", "stale").
// Used identically across kinematics, API serialization, and dashboard metrics.
func ClassifyFlightStatus(lat0, lon0, speedKts, altFt, vertRateFpm float64, onGround bool, route *models.RouteInfo, elapsedSec float64, baseMaxAgeSec, cruiseMaxAgeSec float64) (status string, isCruise bool, effectiveMaxAge float64) {
	if baseMaxAgeSec <= 0 {
		baseMaxAgeSec = 330.0
	}
	if cruiseMaxAgeSec <= 0 {
		cruiseMaxAgeSec = 720.0
	}
	if elapsedSec < 0 {
		elapsedSec = 0
	}

	isCruise = altFt >= 25000.0 && math.Abs(vertRateFpm) < 250.0 && !isTerminalPhase(lat0, lon0, route)
	effectiveMaxAge = baseMaxAgeSec
	if isCruise {
		effectiveMaxAge = cruiseMaxAgeSec
	}

	if elapsedSec > effectiveMaxAge {
		return "stale", isCruise, effectiveMaxAge
	}

	if elapsedSec <= FreshObservationSec || onGround {
		return "observed", isCruise, effectiveMaxAge
	}

	// Airborne dead reckoning forbidden (low speed < 50kt or terminal arrival): coordinates not advanced, reported as observed
	if speedKts > 0 && (speedKts < 50.0 || (altFt > 0 && altFt < 10000.0 && isTerminalPhase(lat0, lon0, route))) {
		return "observed", isCruise, effectiveMaxAge
	}

	return "estimated", isCruise, effectiveMaxAge
}

// ExtrapolateWithCruise advances position with phase-adaptive extrapolation horizons:
// - In cruise (FL250+, level flight, not near terminal): relaxed up to cruiseMaxAgeSec (default 12 minutes).
// - In approach/terminal/climb: strictly clamped to baseMaxAgeSec (default ~5.5 minutes).
// - If elapsedSec <= 30s: status is "observed" (coordinates unchanged).
// - If 30s < elapsedSec <= effectiveMaxAge: status is "estimated" (forward dead reckoning).
// - If elapsedSec > effectiveMaxAge: status is "stale" (clamped, frozen).
func ExtrapolateWithCruise(fd *models.FlightData, now time.Time, baseMaxAgeSec float64, cruiseMaxAgeSec float64) {
	if fd == nil {
		return
	}
	if baseMaxAgeSec <= 0 {
		baseMaxAgeSec = 330.0 // 5.5 minutes default
	}
	if cruiseMaxAgeSec <= 0 {
		cruiseMaxAgeSec = 720.0 // 12 minutes default
	}

	// 1. Determine the timestamp of the last true sensor observation
	var lastObservedEpoch int64
	if fd.Metadata.LastObservedAt > 0 {
		lastObservedEpoch = fd.Metadata.LastObservedAt
	} else if fd.Telemetry.LastContactEpoch > 0 {
		lastObservedEpoch = fd.Telemetry.LastContactEpoch
	} else if !fd.Metadata.LastUpdated.IsZero() {
		lastObservedEpoch = fd.Metadata.LastUpdated.Unix()
	} else {
		return
	}

	lastObservedTime := time.Unix(lastObservedEpoch, 0).UTC()
	elapsedSec := now.Sub(lastObservedTime).Seconds()
	if elapsedSec < 0 {
		elapsedSec = 0
	}

	lat0 := fd.Telemetry.Position.Latitude
	lon0 := fd.Telemetry.Position.Longitude
	speedKts := fd.Telemetry.Speed.GroundSpeedKts
	altFt := fd.Telemetry.Altitude.BarometricFt
	vertRateFpm := fd.Telemetry.Attitude.VerticalRateFpm

	status, _, effectiveMaxAge := ClassifyFlightStatus(
		lat0, lon0, speedKts, altFt, vertRateFpm,
		fd.Telemetry.OnGround,
		&fd.Route, elapsedSec, baseMaxAgeSec, cruiseMaxAgeSec,
	)

	// If timestamp is fresh (within FreshObservationSec)
	if status == "observed" {
		setPositionFields(fd, "observed", int(elapsedSec), lastObservedEpoch)
		return
	}

	// Conditions where dead reckoning is strictly forbidden:
	// a. Aircraft is on the ground
	// b. Ground speed is below 50 knots (stationary, taxiing, or hover)
	// c. Terminal phase: Below 10,000 ft AND within 30 km of departure or destination airport
	if fd.Telemetry.OnGround || speedKts < 50.0 || (altFt < 10000.0 && isTerminalPhase(lat0, lon0, &fd.Route)) {
		setPositionFields(fd, status, int(elapsedSec), lastObservedEpoch)
		if status != "observed" {
			if fd.DiversionAnalysis.PredictedDiversion || fd.DiversionAnalysis.StatusTier == "confirmed" || fd.DiversionAnalysis.StatusTier == "suspected" {
				fd.DiversionAnalysis.PredictedDiversion = false
				fd.DiversionAnalysis.StatusTier = "unverified"
				fd.DiversionAnalysis.ProbabilityLevel = "NONE"
			}
		}
		return
	}

	// 3. Extrapolate forward
	// Clamp extrapolation progression to effectiveMaxAge (frozen if stale)
	effectiveSec := math.Min(elapsedSec, effectiveMaxAge)

	headingDeg := fd.Telemetry.Attitude.TrackDeg

	// Optional high-altitude wind correction hook (NOAA GFS)
	if WindCorrectionHook != nil && altFt >= 18000.0 {
		dSpd, dTrk := WindCorrectionHook(lat0, lon0, altFt, speedKts, headingDeg)
		speedKts = math.Max(50.0, speedKts+dSpd)
		headingDeg = math.Mod(headingDeg+dTrk+360.0, 360.0)
	}

	// Distance traveled in kilometers
	distKm := (speedKts * KnotsToKmh) * (effectiveSec / 3600.0)

	// Destination great-circle curvature bending:
	// If destination coordinates are known, check deviation from bearing to destination.
	// Only bend along great-circle route if current track deviation is less than 30 degrees.
	if fd.Route.Destination != nil && (fd.Route.Destination.Location.Latitude != 0 || fd.Route.Destination.Location.Longitude != 0) {
		destLat := fd.Route.Destination.Location.Latitude
		destLon := fd.Route.Destination.Location.Longitude
		bearingToDest := CalculateBearing(lat0, lon0, destLat, destLon)
		deviation := AngularDifference(headingDeg, bearingToDest)
		if deviation < 30.0 {
			headingDeg = BlendBearing(headingDeg, bearingToDest, 0.3)
		}
	} else if headingDeg == 0 && fd.Route.Destination != nil {
		headingDeg = CalculateBearing(lat0, lon0, fd.Route.Destination.Location.Latitude, fd.Route.Destination.Location.Longitude)
	}

	// Spherical geodesy forward projection (Great Circle)
	lat0Rad := lat0 * math.Pi / 180.0
	lon0Rad := lon0 * math.Pi / 180.0
	headingRad := headingDeg * math.Pi / 180.0
	angularDist := distKm / EarthRadiusKm

	sinLat0 := math.Sin(lat0Rad)
	cosLat0 := math.Cos(lat0Rad)
	sinDist := math.Sin(angularDist)
	cosDist := math.Cos(angularDist)

	lat1Rad := math.Asin(sinLat0*cosDist + cosLat0*sinDist*math.Cos(headingRad))
	lon1Rad := lon0Rad + math.Atan2(math.Sin(headingRad)*sinDist*cosLat0, cosDist-sinLat0*math.Sin(lat1Rad))

	lat1 := lat1Rad * 180.0 / math.Pi
	lon1 := lon1Rad * 180.0 / math.Pi

	// Normalize longitude to [-180, 180]
	for lon1 > 180.0 {
		lon1 -= 360.0
	}
	for lon1 < -180.0 {
		lon1 += 360.0
	}

	// Vertical rate extrapolation with exponential decay over time towards level flight
	newAltFt := altFt
	if vertRateFpm != 0 {
		decay := math.Exp(-effectiveSec / 120.0) // decay factor over time
		deltaAlt := (vertRateFpm * decay) * (effectiveSec / 60.0)
		newAltFt = math.Max(0, math.Min(45000, newAltFt+deltaAlt)) // Clamped to [0, 45000] ft
	}

	// Update telemetry with extrapolated coordinates
	fd.Telemetry.Position.Latitude = math.Round(lat1*100000) / 100000
	fd.Telemetry.Position.Longitude = math.Round(lon1*100000) / 100000
	fd.Telemetry.Altitude.BarometricFt = math.Round(newAltFt)
	fd.Telemetry.Altitude.BarometricM = math.Round(newAltFt * 0.3048)
	fd.Metadata.LastUpdated = now

	// Mark status as estimated or stale
	setPositionFields(fd, status, int(elapsedSec), lastObservedEpoch)
	if status != "observed" {
		if fd.DiversionAnalysis.PredictedDiversion || fd.DiversionAnalysis.StatusTier == "confirmed" || fd.DiversionAnalysis.StatusTier == "suspected" {
			fd.DiversionAnalysis.PredictedDiversion = false
			fd.DiversionAnalysis.StatusTier = "unverified"
			fd.DiversionAnalysis.ProbabilityLevel = "NONE"
		}
	}
}

func setPositionFields(fd *models.FlightData, source string, ageSec int, lastObs int64) {
	fd.PositionSource = source
	fd.EstimatedAgeS = ageSec
	fd.LastObservedAt = lastObs

	fd.Telemetry.PositionSource = source
	fd.Telemetry.EstimatedAgeS = ageSec
	fd.Telemetry.LastObservedAt = lastObs

	fd.Metadata.PositionSource = source
	fd.Metadata.EstimatedAgeS = ageSec
	fd.Metadata.LastObservedAt = lastObs
}

func isTerminalPhase(lat, lon float64, route *models.RouteInfo) bool {
	if route == nil {
		return false
	}
	// Check departure airport
	if route.Origin != nil && (route.Origin.Location.Latitude != 0 || route.Origin.Location.Longitude != 0) {
		d := HaversineDistance(lat, lon, route.Origin.Location.Latitude, route.Origin.Location.Longitude)
		if d < 30.0 {
			return true
		}
	}
	// Check destination airport
	if route.Destination != nil && (route.Destination.Location.Latitude != 0 || route.Destination.Location.Longitude != 0) {
		d := HaversineDistance(lat, lon, route.Destination.Location.Latitude, route.Destination.Location.Longitude)
		if d < 30.0 {
			return true
		}
	}
	return false
}

// CalculateBearing calculates the initial bearing in degrees from point 1 to point 2.
func CalculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180.0
	lon1Rad := lon1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	lon2Rad := lon2 * math.Pi / 180.0

	dLon := lon2Rad - lon1Rad
	y := math.Sin(dLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLon)

	bearingRad := math.Atan2(y, x)
	bearingDeg := bearingRad * 180.0 / math.Pi
	return math.Mod(bearingDeg+360.0, 360.0)
}

// HaversineDistance calculates great circle distance in kilometers.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusKm * c
}

// AngularDifference returns the absolute difference in degrees between two angles [0, 180].
func AngularDifference(deg1, deg2 float64) float64 {
	diff := math.Mod(math.Abs(deg1-deg2), 360.0)
	if diff > 180.0 {
		diff = 360.0 - diff
	}
	return diff
}

// BlendBearing interpolates bearing angle b1 towards b2 by factor t [0, 1].
func BlendBearing(b1, b2, t float64) float64 {
	diff := math.Mod(b2-b1+540.0, 360.0) - 180.0
	res := math.Mod(b1+diff*t+360.0, 360.0)
	return res
}
