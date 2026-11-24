package models

import (
	"time"
)

// AircraftInfo contains aircraft identity and type details.
type AircraftInfo struct {
	Hex          string `json:"hex"`
	Registration string `json:"registration,omitempty"`
	Type         string `json:"type,omitempty"`
	Category     string `json:"category,omitempty"`
	Country      string `json:"country,omitempty"`
}

// AirlineInfo contains the airline operator details.
type AirlineInfo struct {
	Name    string `json:"name,omitempty"`
	ICAO    string `json:"icao,omitempty"`
	IATA    string `json:"iata,omitempty"`
	Country string `json:"country,omitempty"`
}

// AirportLocation contains geographic coordinates of an airport.
type AirportLocation struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	ElevationFt int     `json:"elevation_ft,omitempty"`
}

// AirportInfo contains detailed information about an airport.
type AirportInfo struct {
	ICAO         string          `json:"icao"`
	IATA         string          `json:"iata,omitempty"`
	Name         string          `json:"name"`
	City         string          `json:"city,omitempty"`
	Country      string          `json:"country,omitempty"`
	Type         string          `json:"type,omitempty"`
	Location     AirportLocation `json:"location"`
	DistanceKm   float64         `json:"distance_km,omitempty"`
	BearingDeg   float64         `json:"bearing_deg,omitempty"`
}

// RouteInfo holds the flight's origin, destination, and route string.
type RouteInfo struct {
	Summary     string       `json:"summary,omitempty"`
	Origin      *AirportInfo `json:"origin,omitempty"`
	Destination *AirportInfo `json:"destination,omitempty"`
}

// PositionInfo holds latitude and longitude.
type PositionInfo struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// AltitudeInfo holds altitude in different units.
type AltitudeInfo struct {
	BarometricFt float64 `json:"barometric_ft"`
	GeometricFt  float64 `json:"geometric_ft,omitempty"`
	BarometricM  float64 `json:"barometric_m"`
}

// SpeedInfo holds speed telemetry.
type SpeedInfo struct {
	GroundSpeedKts float64 `json:"ground_speed_kts"`
	GroundSpeedKmh float64 `json:"ground_speed_kmh"`
}

// AttitudeInfo holds aircraft heading and vertical rate.
type AttitudeInfo struct {
	TrackDeg        float64 `json:"track_deg"`
	VerticalRateFpm float64 `json:"vertical_rate_fpm"`
	VerticalRateMps float64 `json:"vertical_rate_mps"`
}

// TelemetryInfo represents real-time flight kinematics.
type TelemetryInfo struct {
	Position         PositionInfo `json:"position"`
	Altitude         AltitudeInfo `json:"altitude"`
	Speed            SpeedInfo    `json:"speed"`
	Attitude         AttitudeInfo `json:"attitude"`
	OnGround         bool         `json:"on_ground"`
	LastContactEpoch int64        `json:"last_contact_epoch"`
	PositionSource   string       `json:"position_source"` // "observed" or "estimated"
	EstimatedAgeS    int          `json:"estimated_age_s"`
	LastObservedAt   int64        `json:"last_observed_at"`
}

// EmergencyCategory defines emergency classifications.
type EmergencyCategory string

const (
	EmergencyNone                EmergencyCategory = "NONE"
	EmergencyGeneral             EmergencyCategory = "GENERAL_EMERGENCY"              // Squawk 7700
	EmergencyRadioFailure        EmergencyCategory = "RADIO_FAILURE"                  // Squawk 7600
	EmergencyHijack              EmergencyCategory = "HIJACK_UNLAWFUL_INTERFERENCE"    // Squawk 7500
	EmergencyUpstreamGeneral     EmergencyCategory = "UPSTREAM_REPORTED_EMERGENCY"
)

// EmergencyInfo details transponder squawk and emergency status.
type EmergencyInfo struct {
	Squawk      string            `json:"squawk"`
	IsEmergency bool              `json:"is_emergency"`
	Category    EmergencyCategory `json:"category"`
	Description string            `json:"description"`
}

// DiversionCriterion details an individual triggered diversion condition.
type DiversionCriterion struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Triggered bool   `json:"triggered"`
}

// AlternateCandidate represents a scored candidate airport for diversion.
type AlternateCandidate struct {
	AirportInfo
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// DiversionAnalysis represents the computed diversion prediction.
type DiversionAnalysis struct {
	PredictedDiversion  bool                 `json:"predicted_diversion"`
	ProbabilityLevel    string               `json:"probability_level"` // "NONE", "LOW", "MODERATE", "HIGH", "CRITICAL"
	StatusTier          string               `json:"status_tier"`       // "none", "unverified", "suspected", "confirmed"
	IsHolding           bool                 `json:"is_holding"`
	TriggerReasons      []string             `json:"trigger_reasons,omitempty"`
	TopCandidates       []AlternateCandidate `json:"top_candidates,omitempty"`
	DistanceToDestKm    float64              `json:"distance_to_destination_km"`
	BearingToDestDeg    float64              `json:"bearing_to_destination_deg"`
	HeadingDeviationDeg float64              `json:"heading_deviation_deg"`
	TriggeredCriteria   []DiversionCriterion `json:"triggered_criteria"`
	CandidateAlternate  *AirportInfo         `json:"candidate_alternate,omitempty"`
}

// MetadataInfo captures data provenance and timestamps.
type MetadataInfo struct {
	UpstreamSource string    `json:"upstream_source"`
	LastUpdated    time.Time `json:"last_updated"`
	PositionSource string    `json:"position_source"` // "observed" or "estimated"
	EstimatedAgeS  int       `json:"estimated_age_s"`
	LastObservedAt int64     `json:"last_observed_at"`
}

// FlightData is the unified model combining identity, route, telemetry, emergency, and diversion analysis.
type FlightData struct {
	Aircraft          AircraftInfo      `json:"aircraft"`
	Flight            FlightIdentifier  `json:"flight"`
	Route             RouteInfo         `json:"route"`
	Telemetry         TelemetryInfo     `json:"telemetry"`
	Emergency         EmergencyInfo     `json:"emergency"`
	DiversionAnalysis DiversionAnalysis `json:"diversion_analysis"`
	Metadata          MetadataInfo      `json:"metadata"`
	PositionSource    string            `json:"position_source"` // "observed" or "estimated"
	EstimatedAgeS     int               `json:"estimated_age_s"`
	LastObservedAt    int64             `json:"last_observed_at"`
}

// UpstreamState tracks provider rate limits, budget, and cooldown across restarts.
type UpstreamState struct {
	Provider            string `json:"provider"`
	Status              string `json:"status"` // "ok", "cooldown", "error"
	LastSuccessAt       int64  `json:"last_success_at"`
	NextAllowedAt       int64  `json:"next_allowed_at"`
	RemainingCredits    int    `json:"remaining_credits"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	CooldownUntil       int64  `json:"cooldown_until"`
	LastFetchedCount    int    `json:"last_fetched_count"`
	UpdatedAt           int64  `json:"updated_at"`
}

// FlightIdentifier holds callsign formats.
type FlightIdentifier struct {
	CallsignICAO string       `json:"callsign_icao"`
	CallsignIATA *string      `json:"callsign_iata"`
	Airline      *AirlineInfo `json:"airline,omitempty"`
}

// RawAircraftSnapshot represents the normalized aircraft state extracted from any upstream provider.
type RawAircraftSnapshot struct {
	Hex          string
	Flight       string // trimmed callsign
	Squawk       string
	Emergency    string
	Lat          float64
	Lon          float64
	AltBaroFt    float64
	AltGeomFt    float64
	GroundSpeed  float64 // knots
	Track        float64 // degrees
	VerticalRate float64 // ft/min
	OnGround     bool
	Category     string
	Type         string
	Registration string
	LastContact  int64
	Source       string
}

// Pagination holds pagination metadata.
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
	TotalItems int `json:"total_items"`
}

// AttributionNotice contains explicit upstream citations and non-commercial terms.
type AttributionNotice struct {
	Notice        string   `json:"notice"`
	Sources       []string `json:"sources"`
	Citation      string   `json:"paper_citation"`
	DataNoticeURL string   `json:"data_notice_url"`
}

// DefaultAttribution returns the standard attribution notice required by upstream data agreements.
func DefaultAttribution(baseURL string) *AttributionNotice {
	return &AttributionNotice{
		Notice: "Strictly Non-Commercial Use Only. Contrail collects and redistributes data as-is. All data remains the property of its upstream providers and is subject to their terms. Not for navigation or safety-of-life use.",
		Sources: []string{
			"Live aircraft positions: The OpenSky Network (https://opensky-network.org)",
			"Live aircraft positions: adsb.lol (https://adsb.lol, licensed under ODbL: https://opendatacommons.org/licenses/odbl/)",
			"Flight routes: adsbdb.com — route data by David Taylor (Edinburgh) and Jim Mason (Glasgow)",
			"Aircraft registrations & types: adsbdb.com, aircraft data from PlaneBase",
			"Airports: OurAirports (https://ourairports.com, public domain)",
		},
		Citation:      "Matthias Schäfer, Martin Strohmeier, Vincent Lenders, Ivan Martinovic and Matthias Wilhelm. \"Bringing Up OpenSky: A Large-scale ADS-B Sensor Network for Research\". In Proceedings of the 13th IEEE/ACM International Symposium on Information Processing in Sensor Networks (IPSN), pages 83-94, April 2014.",
		DataNoticeURL: baseURL + "/data-notice",
	}
}

// APIResponse is the standard JSON envelope for API responses.
type APIResponse struct {
	APIVersion    string             `json:"api_version"`
	BaseURL       string             `json:"base_url"`
	GeneratedAt   time.Time          `json:"generated_at"`
	ExecutionMs   float64            `json:"execution_ms"`
	Status        string             `json:"status"` // "success", "error"
	ErrorMessage  string             `json:"error_message,omitempty"`
	Attribution   *AttributionNotice `json:"attribution,omitempty"`
	TotalRecords  int                `json:"total_records"`
	FilteredCount int                `json:"filtered_count"`
	Pagination    *Pagination        `json:"pagination,omitempty"`
	Data          interface{}        `json:"data"`
}

// BoundingBoxFilter specifies geographic bounds.
type BoundingBoxFilter struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

// FlightQueryFilter is the custom JSON query request format.
type FlightQueryFilter struct {
	CallsignPrefix   string             `json:"callsign_prefix,omitempty"`
	Hex              string             `json:"hex,omitempty"`
	IsEmergency      *bool              `json:"is_emergency,omitempty"`
	IsDiverting      *bool              `json:"is_diverting,omitempty"`
	Squawk           string             `json:"squawk,omitempty"`
	OriginICAO       string             `json:"origin_icao,omitempty"`
	OriginIATA       string             `json:"origin_iata,omitempty"`
	DestinationICAO  string             `json:"destination_icao,omitempty"`
	DestinationIATA  string             `json:"destination_iata,omitempty"`
	MinAltitudeFt    *float64           `json:"min_altitude_ft,omitempty"`
	MaxAltitudeFt    *float64           `json:"max_altitude_ft,omitempty"`
	OnGround         *bool              `json:"on_ground,omitempty"`
	BoundingBox      *BoundingBoxFilter `json:"bounding_box,omitempty"`
}

// QueryRequest is the top-level payload for POST /api/v1/query.
type QueryRequest struct {
	Filters   FlightQueryFilter `json:"filters"`
	SortBy    string            `json:"sort_by,omitempty"`    // "last_seen", "altitude", "speed", "callsign"
	SortOrder string            `json:"sort_order,omitempty"` // "asc", "desc"
	Limit     int               `json:"limit,omitempty"`
	Page      int               `json:"page,omitempty"`
}
