package weather

import (
	"math"
	"sync"
	"time"

	"contrail/internal/config"
)

// WindVector represents horizontal wind components.
// U is eastward velocity component in knots, V is northward velocity component in knots.
type WindVector struct {
	U            float64 `json:"u_component_kts"`
	V            float64 `json:"v_component_kts"`
	SpeedKts     float64 `json:"speed_kts"`
	DirectionDeg float64 `json:"direction_deg"`
}

// WindFieldProvider defines the interface for retrieving high-altitude winds (e.g. NOAA GFS 300/250 hPa).
type WindFieldProvider interface {
	GetWind(lat, lon, altFt float64) (WindVector, bool)
}

// ConvectiveRegion defines a severe convective/thunderstorm cell area.
type ConvectiveRegion struct {
	ID        string    `json:"id"`
	CenterLat float64   `json:"center_lat"`
	CenterLon float64   `json:"center_lon"`
	RadiusKm  float64   `json:"radius_km"`
	Severity  string    `json:"severity"` // "MODERATE", "SEVERE"
	CloudTopFt float64  `json:"cloud_top_ft"`
	ValidUntil time.Time `json:"valid_until"`
}

// ConvectiveWeatherProvider provides convective thunderstorm regions to avoid false positive diversion scoring.
type ConvectiveWeatherProvider interface {
	IsConvectiveAvoidance(lat, lon, headingDeg float64) (bool, string)
	GetActiveCells() []ConvectiveRegion
}

// NOAAGFSWindProvider implements a memory-cached coarse grid (e.g. 1.0 degree isobaric 250/300 hPa) wind model.
type NOAAGFSWindProvider struct {
	mu        sync.RWMutex
	enabled   bool
	gridCache map[int]WindVector // key: latIdx*1000 + lonIdx
	lastFetch time.Time
}

func NewNOAAGFSWindProvider(cfg *config.Config) *NOAAGFSWindProvider {
	return &NOAAGFSWindProvider{
		enabled:   cfg.WeatherWindsEnabled,
		gridCache: make(map[int]WindVector),
	}
}

// GetWind retrieves wind vector for the specified location and altitude.
// Default disabled returns (zero, false).
func (p *NOAAGFSWindProvider) GetWind(lat, lon, altFt float64) (WindVector, bool) {
	if !p.enabled {
		return WindVector{}, false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	latIdx := int(math.Floor(lat))
	lonIdx := int(math.Floor(lon))
	key := latIdx*1000 + lonIdx

	if w, ok := p.gridCache[key]; ok {
		return w, true
	}

	return WindVector{}, false
}

// NOAAConvectiveProvider implements convective storm detection interface.
type NOAAConvectiveProvider struct {
	mu      sync.RWMutex
	enabled bool
	cells   []ConvectiveRegion
}

func NewNOAAConvectiveProvider(cfg *config.Config) *NOAAConvectiveProvider {
	return &NOAAConvectiveProvider{
		enabled: cfg.WeatherRadarLayerEnabled,
	}
}

func (p *NOAAConvectiveProvider) IsConvectiveAvoidance(lat, lon, headingDeg float64) (bool, string) {
	if !p.enabled {
		return false, ""
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	for _, cell := range p.cells {
		if cell.ValidUntil.Before(now) {
			continue
		}
		// Haversine distance to storm cell
		dist := haversineKm(lat, lon, cell.CenterLat, cell.CenterLon)
		if dist <= cell.RadiusKm+30.0 { // Within cell or 30km safety buffer
			return true, cell.Severity
		}
	}

	return false, ""
}

func (p *NOAAConvectiveProvider) GetActiveCells() []ConvectiveRegion {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]ConvectiveRegion(nil), p.cells...)
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	rLat1 := lat1 * math.Pi / 180.0
	rLat2 := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}
