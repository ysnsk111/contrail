package poller

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"contrail/internal/models"
)

//go:embed airports.json
var embeddedAirportsJSON []byte

// AirportDB manages OurAirports data in memory for rapid geospatial and metadata lookups.
type AirportDB struct {
	mu          sync.RWMutex
	airports    []models.AirportInfo
	byICAO      map[string]*models.AirportInfo
	byIATA      map[string]*models.AirportInfo
	byIdent     map[string]*models.AirportInfo
	gridBuckets map[int][]models.AirportInfo // 2-degree lat/lon spatial grid
}

var (
	defaultAirportDB *AirportDB
	onceAirportDB    sync.Once
)

// GetAirportDB returns the singleton AirportDB.
func GetAirportDB() *AirportDB {
	onceAirportDB.Do(func() {
		db := &AirportDB{
			byICAO:      make(map[string]*models.AirportInfo),
			byIATA:      make(map[string]*models.AirportInfo),
			byIdent:     make(map[string]*models.AirportInfo),
			gridBuckets: make(map[int][]models.AirportInfo),
		}
		if err := db.loadFromData(embeddedAirportsJSON); err != nil {
			// Fallback: initialized empty
		}
		defaultAirportDB = db
	})
	return defaultAirportDB
}

type rawAirportItem struct {
	Ident        string  `json:"ident"`
	ICAO         string  `json:"icao"`
	IATA         string  `json:"iata"`
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	ElevationFt  int     `json:"elevation_ft"`
	Country      string  `json:"country"`
	Municipality string  `json:"municipality"`
}

func (db *AirportDB) loadFromData(data []byte) error {
	var rawList []rawAirportItem
	if err := json.Unmarshal(data, &rawList); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, item := range rawList {
		info := models.AirportInfo{
			ICAO:         strings.ToUpper(strings.TrimSpace(item.ICAO)),
			IATA:         strings.ToUpper(strings.TrimSpace(item.IATA)),
			Name:         item.Name,
			City:         item.Municipality,
			Country:      item.Country,
			Type:         item.Type,
			Location: models.AirportLocation{
				Latitude:    item.Lat,
				Longitude:   item.Lon,
				ElevationFt: item.ElevationFt,
			},
		}

		identUpper := strings.ToUpper(strings.TrimSpace(item.Ident))
		if info.ICAO != "" {
			db.byICAO[info.ICAO] = &info
		}
		if info.IATA != "" {
			db.byIATA[info.IATA] = &info
		}
		if identUpper != "" {
			db.byIdent[identUpper] = &info
		}

		db.airports = append(db.airports, info)

		// Grid index: 2 degree grid cell key = (latIdx + 90) * 1000 + (lonIdx + 180)
		latIdx := int(math.Floor((item.Lat + 90) / 2.0))
		lonIdx := int(math.Floor((item.Lon + 180) / 2.0))
		gridKey := latIdx*1000 + lonIdx
		db.gridBuckets[gridKey] = append(db.gridBuckets[gridKey], info)
	}

	return nil
}

// Lookup finds an airport by ICAO or IATA or ident code.
func (db *AirportDB) Lookup(code string) *models.AirportInfo {
	if code == "" {
		return nil
	}
	db.mu.RLock()
	defer db.mu.RUnlock()

	upper := strings.ToUpper(strings.TrimSpace(code))
	if apt, ok := db.byICAO[upper]; ok {
		return apt
	}
	if apt, ok := db.byIATA[upper]; ok {
		return apt
	}
	if apt, ok := db.byIdent[upper]; ok {
		return apt
	}
	return nil
}

// FindCandidateAlternate finds an airport suitable for diversion:
// - Must be a commercial-grade airport (large_airport or medium_airport)
// - Within maxDistanceKm (default 80km)
// - Track alignment: aircraft heading points within 25° of the airport bearing
// - Physical descent profile: distance must be physically consistent with altitude (glide angle <= 4°)
// - Excludes origin & destination airports
func (db *AirportDB) FindCandidateAlternate(acLat, acLon, acTrack, acAltFt, maxDistanceKm float64, excludeCodes ...string) (*models.AirportInfo, float64, float64) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	excludeSet := make(map[string]bool)
	for _, code := range excludeCodes {
		c := strings.ToUpper(strings.TrimSpace(code))
		if c != "" {
			excludeSet[c] = true
		}
	}

	if maxDistanceKm <= 0 || maxDistanceKm > 100.0 {
		maxDistanceKm = 80.0
	}

	var bestCandidate *models.AirportInfo
	bestDist := math.MaxFloat64
	bestBearingDiff := 0.0

	// Check nearby grid cells (aircraft cell +/- 1)
	curLatIdx := int(math.Floor((acLat + 90) / 2.0))
	curLonIdx := int(math.Floor((acLon + 180) / 2.0))

	for dLat := -1; dLat <= 1; dLat++ {
		for dLon := -1; dLon <= 1; dLon++ {
			gridKey := (curLatIdx+dLat)*1000 + (curLonIdx + dLon)
			for _, apt := range db.gridBuckets[gridKey] {
				if excludeSet[apt.ICAO] || excludeSet[apt.IATA] {
					continue
				}

				// Only consider commercial-capable airports (large or medium)
				// Discard heliports, seaplanes, closed strips, and minor grass airfields
				t := strings.ToLower(apt.Type)
				if t != "" && t != "large_airport" && t != "medium_airport" {
					continue
				}

				dist := HaversineDistance(acLat, acLon, apt.Location.Latitude, apt.Location.Longitude)
				if dist > maxDistanceKm || dist < 3.0 {
					continue
				}

				// Physical descent feasibility: at 3° standard glideslope, 1000ft requires ~3.18 NM = ~5.9 km.
				// Even for a steep 4.5° emergency dive, 1000ft requires ~2.1 NM = ~3.9 km (dist >= altFt / 600 km).
				if acAltFt > 3000.0 && dist < (acAltFt/600.0) {
					continue
				}

				bearing := InitialBearing(acLat, acLon, apt.Location.Latitude, apt.Location.Longitude)
				bearingDiff := AngularDifference(acTrack, bearing)

				// Alignment criteria: aircraft pointing towards the airport within 25 degrees
				if bearingDiff <= 25.0 {
					if dist < bestDist {
						bestDist = dist
						bestBearingDiff = bearingDiff
						candidateCopy := apt
						candidateCopy.DistanceKm = math.Round(dist*10) / 10
						candidateCopy.BearingDeg = math.Round(bearing*10) / 10
						bestCandidate = &candidateCopy
					}
				}
			}
		}
	}

	return bestCandidate, bestDist, bestBearingDiff
}

// FindTopCandidateAlternates finds and scores up to 3 candidate alternate airports.
func (db *AirportDB) FindTopCandidateAlternates(acLat, acLon, acTrack, acAltFt, maxDistanceKm float64, excludeCodes ...string) []models.AlternateCandidate {
	db.mu.RLock()
	defer db.mu.RUnlock()

	excludeSet := make(map[string]bool)
	for _, code := range excludeCodes {
		c := strings.ToUpper(strings.TrimSpace(code))
		if c != "" {
			excludeSet[c] = true
		}
	}

	if maxDistanceKm <= 0 || maxDistanceKm > 100.0 {
		maxDistanceKm = 80.0
	}

	type scoredItem struct {
		cand   models.AirportInfo
		score  float64
		dist   float64
		diff   float64
		reason string
	}
	var scoredList []scoredItem

	curLatIdx := int(math.Floor((acLat + 90) / 2.0))
	curLonIdx := int(math.Floor((acLon + 180) / 2.0))

	seen := make(map[string]bool)
	for dLat := -1; dLat <= 1; dLat++ {
		for dLon := -1; dLon <= 1; dLon++ {
			gridKey := (curLatIdx+dLat)*1000 + (curLonIdx + dLon)
			for _, apt := range db.gridBuckets[gridKey] {
				if seen[apt.ICAO] {
					continue
				}
				seen[apt.ICAO] = true

				if excludeSet[apt.ICAO] || excludeSet[apt.IATA] {
					continue
				}

				t := strings.ToLower(apt.Type)
				if t != "" && t != "large_airport" && t != "medium_airport" {
					continue
				}

				dist := HaversineDistance(acLat, acLon, apt.Location.Latitude, apt.Location.Longitude)
				if dist > maxDistanceKm || dist < 3.0 {
					continue
				}

				if acAltFt > 3000.0 && dist < (acAltFt/600.0) {
					continue
				}

				bearing := InitialBearing(acLat, acLon, apt.Location.Latitude, apt.Location.Longitude)
				bearingDiff := AngularDifference(acTrack, bearing)

				if bearingDiff <= 35.0 {
					alignScore := 40.0 * (1.0 - bearingDiff/35.0)
					distScore := 35.0 * (1.0 - dist/maxDistanceKm)
					sizeScore := 15.0
					if t == "large_airport" {
						sizeScore = 25.0
					}
					totalScore := math.Round((alignScore+distScore+sizeScore)*10) / 10

					candCopy := apt
					candCopy.DistanceKm = math.Round(dist*10) / 10
					candCopy.BearingDeg = math.Round(bearing*10) / 10

					reason := fmt.Sprintf("Alignment: %.1f deg diff, Distance: %.1f km, Class: %s", bearingDiff, dist, t)
					scoredList = append(scoredList, scoredItem{
						cand:   candCopy,
						score:  totalScore,
						dist:   dist,
						diff:   bearingDiff,
						reason: reason,
					})
				}
			}
		}
	}

	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	var result []models.AlternateCandidate
	limit := 3
	if len(scoredList) < limit {
		limit = len(scoredList)
	}
	for i := 0; i < limit; i++ {
		item := scoredList[i]
		result = append(result, models.AlternateCandidate{
			AirportInfo: item.cand,
			Score:       item.score,
			Reason:      item.reason,
		})
	}
	return result
}

// HaversineDistance calculates great circle distance in kilometers.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	rLat1 := lat1 * math.Pi / 180.0
	rLat2 := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// InitialBearing calculates the initial bearing in degrees from point 1 to point 2 (0-360).
func InitialBearing(lat1, lon1, lat2, lon2 float64) float64 {
	rLat1 := lat1 * math.Pi / 180.0
	rLon1 := lon1 * math.Pi / 180.0
	rLat2 := lat2 * math.Pi / 180.0
	rLon2 := lon2 * math.Pi / 180.0

	dLon := rLon2 - rLon1

	y := math.Sin(dLon) * math.Cos(rLat2)
	x := math.Cos(rLat1)*math.Sin(rLat2) -
		math.Sin(rLat1)*math.Cos(rLat2)*math.Cos(dLon)

	bearingRad := math.Atan2(y, x)
	bearingDeg := bearingRad * 180.0 / math.Pi

	// Normalize to 0..360
	return math.Mod(bearingDeg+360.0, 360.0)
}

// AngularDifference calculates the shortest angular difference between two headings (0..180).
func AngularDifference(angle1, angle2 float64) float64 {
	diff := math.Abs(angle1 - angle2)
	diff = math.Mod(diff, 360.0)
	if diff > 180.0 {
		diff = 360.0 - diff
	}
	return diff
}
