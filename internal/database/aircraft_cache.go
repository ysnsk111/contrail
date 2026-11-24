package database

import (
	"database/sql"
	"time"
)

// CachedAircraft holds cached registration, type, and model.
type CachedAircraft struct {
	Hex          string
	Registration string
	TypeCode     string
	Model        string
	Country      string
	CachedAt     time.Time
	ExpiresAt    time.Time
}

// GetCachedAircraft retrieves cached aircraft metadata if not expired.
func (db *DB) GetCachedAircraft(hex string) (*CachedAircraft, bool, error) {
	nowUnix := time.Now().Unix()
	query := `
		SELECT hex, registration, typecode, model, country, cached_at, expires_at
		FROM aircraft_cache
		WHERE hex = ? AND expires_at > ?
	`
	row := db.QueryRow(query, hex, nowUnix)

	var (
		h, reg, tCode, model, country sql.NullString
		cachedAt, expiresAt           int64
	)

	err := row.Scan(&h, &reg, &tCode, &model, &country, &cachedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return &CachedAircraft{
		Hex:          h.String,
		Registration: reg.String,
		TypeCode:     tCode.String,
		Model:        model.String,
		Country:      country.String,
		CachedAt:     time.Unix(cachedAt, 0),
		ExpiresAt:    time.Unix(expiresAt, 0),
	}, true, nil
}

// SetCachedAircraft stores or updates aircraft metadata in the database cache.
func (db *DB) SetCachedAircraft(hex, registration, typecode, model, country string, ttl time.Duration) error {
	now := time.Now()
	expires := now.Add(ttl)

	stmt := `
		INSERT INTO aircraft_cache (hex, registration, typecode, model, country, cached_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hex) DO UPDATE SET
			registration=excluded.registration,
			typecode=excluded.typecode,
			model=excluded.model,
			country=excluded.country,
			cached_at=excluded.cached_at,
			expires_at=excluded.expires_at;
	`
	_, err := db.Exec(stmt, hex, registration, typecode, model, country, now.Unix(), expires.Unix())
	return err
}
