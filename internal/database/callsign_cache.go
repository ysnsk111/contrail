package database

import (
	"database/sql"
	"time"

	"contrail/internal/models"
)

// CachedRoute holds cached route and airline info.
type CachedRoute struct {
	Callsign   string
	Status     string // "found" or "unknown"
	Route      *models.RouteInfo
	Airline    *models.AirlineInfo
	IATA       string
	CachedAt   time.Time
	ExpiresAt  time.Time
}

// GetCachedCallsign retrieves route info if present.
func (db *DB) GetCachedCallsign(callsign string) (*CachedRoute, bool, error) {
	query := `
		SELECT callsign, callsign_iata, airline_name, airline_icao, airline_iata, airline_country,
		       origin_icao, origin_iata, origin_name, origin_city, origin_country, origin_lat, origin_lon, origin_elev,
		       dest_icao, dest_iata, dest_name, dest_city, dest_country, dest_lat, dest_lon, dest_elev,
		       route_summary, status, cached_at, expires_at
		FROM callsign_cache
		WHERE callsign = ?
		ORDER BY cached_at DESC LIMIT 1
	`
	row := db.QueryRow(query, callsign)

	var (
		cs, csIata, airName, airIcao, airIata, airCountry                 sql.NullString
		origIcao, origIata, origName, origCity, origCountry              sql.NullString
		origLat, origLon                                                 sql.NullFloat64
		origElev                                                         sql.NullInt64
		destIcao, destIata, destName, destCity, destCountry              sql.NullString
		destLat, destLon                                                 sql.NullFloat64
		destElev                                                         sql.NullInt64
		summary, status                                                  sql.NullString
		cachedAt, expiresAt                                              int64
	)

	err := row.Scan(
		&cs, &csIata, &airName, &airIcao, &airIata, &airCountry,
		&origIcao, &origIata, &origName, &origCity, &origCountry, &origLat, &origLon, &origElev,
		&destIcao, &destIata, &destName, &destCity, &destCountry, &destLat, &destLon, &destElev,
		&summary, &status, &cachedAt, &expiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	cached := &CachedRoute{
		Callsign:  cs.String,
		Status:    status.String,
		IATA:      csIata.String,
		CachedAt:  time.Unix(cachedAt, 0),
		ExpiresAt: time.Unix(expiresAt, 0),
	}

	if status.String == "found" {
		cached.Airline = &models.AirlineInfo{
			Name:    airName.String,
			ICAO:    airIcao.String,
			IATA:    airIata.String,
			Country: airCountry.String,
		}

		route := &models.RouteInfo{
			Summary: summary.String,
		}

		if origIcao.Valid && origIcao.String != "" {
			route.Origin = &models.AirportInfo{
				ICAO:    origIcao.String,
				IATA:    origIata.String,
				Name:    origName.String,
				City:    origCity.String,
				Country: origCountry.String,
				Location: models.AirportLocation{
					Latitude:    origLat.Float64,
					Longitude:   origLon.Float64,
					ElevationFt: int(origElev.Int64),
				},
			}
		}

		if destIcao.Valid && destIcao.String != "" {
			route.Destination = &models.AirportInfo{
				ICAO:    destIcao.String,
				IATA:    destIata.String,
				Name:    destName.String,
				City:    destCity.String,
				Country: destCountry.String,
				Location: models.AirportLocation{
					Latitude:    destLat.Float64,
					Longitude:   destLon.Float64,
					ElevationFt: int(destElev.Int64),
				},
			}
		}
		cached.Route = route
	}

	return cached, true, nil
}

// SetCachedCallsign stores or updates route cache entry.
func (db *DB) SetCachedCallsign(callsign string, iata string, airline *models.AirlineInfo, route *models.RouteInfo, status string, ttl time.Duration) error {
	now := time.Now()
	expires := now.Add(ttl)

	stmt := `
		INSERT INTO callsign_cache (
			callsign, callsign_iata, airline_name, airline_icao, airline_iata, airline_country,
			origin_icao, origin_iata, origin_name, origin_city, origin_country, origin_lat, origin_lon, origin_elev,
			dest_icao, dest_iata, dest_name, dest_city, dest_country, dest_lat, dest_lon, dest_elev,
			route_summary, status, cached_at, expires_at
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?
		)
		ON CONFLICT(callsign) DO UPDATE SET
			callsign_iata=excluded.callsign_iata,
			airline_name=excluded.airline_name,
			airline_icao=excluded.airline_icao,
			airline_iata=excluded.airline_iata,
			airline_country=excluded.airline_country,
			origin_icao=excluded.origin_icao,
			origin_iata=excluded.origin_iata,
			origin_name=excluded.origin_name,
			origin_city=excluded.origin_city,
			origin_country=excluded.origin_country,
			origin_lat=excluded.origin_lat,
			origin_lon=excluded.origin_lon,
			origin_elev=excluded.origin_elev,
			dest_icao=excluded.dest_icao,
			dest_iata=excluded.dest_iata,
			dest_name=excluded.dest_name,
			dest_city=excluded.dest_city,
			dest_country=excluded.dest_country,
			dest_lat=excluded.dest_lat,
			dest_lon=excluded.dest_lon,
			dest_elev=excluded.dest_elev,
			route_summary=excluded.route_summary,
			status=excluded.status,
			cached_at=excluded.cached_at,
			expires_at=excluded.expires_at;
	`

	var (
		airName, airIcao, airIata, airCountry    interface{}
		origIcao, origIata, origName, origCity, origCountry interface{}
		origLat, origLon, origElev               interface{}
		destIcao, destIata, destName, destCity, destCountry interface{}
		destLat, destLon, destElev               interface{}
		summary                                  interface{}
	)

	if airline != nil {
		airName = airline.Name
		airIcao = airline.ICAO
		airIata = airline.IATA
		airCountry = airline.Country
	}

	if route != nil {
		summary = route.Summary
		if route.Origin != nil {
			origIcao = route.Origin.ICAO
			origIata = route.Origin.IATA
			origName = route.Origin.Name
			origCity = route.Origin.City
			origCountry = route.Origin.Country
			origLat = route.Origin.Location.Latitude
			origLon = route.Origin.Location.Longitude
			origElev = route.Origin.Location.ElevationFt
		}
		if route.Destination != nil {
			destIcao = route.Destination.ICAO
			destIata = route.Destination.IATA
			destName = route.Destination.Name
			destCity = route.Destination.City
			destCountry = route.Destination.Country
			destLat = route.Destination.Location.Latitude
			destLon = route.Destination.Location.Longitude
			destElev = route.Destination.Location.ElevationFt
		}
	}

	_, err := db.Exec(stmt,
		callsign, iata, airName, airIcao, airIata, airCountry,
		origIcao, origIata, origName, origCity, origCountry, origLat, origLon, origElev,
		destIcao, destIata, destName, destCity, destCountry, destLat, destLon, destElev,
		summary, status, now.Unix(), expires.Unix(),
	)
	return err
}
