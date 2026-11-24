package database

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

// CorridorSegment represents a cross-track waypoint along the flight journey percentage.
type CorridorSegment struct {
	ProgressPercent float64 `json:"progress_percent"` // 0.0 to 1.0
	CenterLat       float64 `json:"center_lat"`
	CenterLon       float64 `json:"center_lon"`
	WidthKm         float64 `json:"width_km"` // 2-sigma corridor width
}

// RouteCorridor represents learned normal flight corridor for an origin-destination pair.
type RouteCorridor struct {
	OriginICAO  string            `json:"origin_icao"`
	DestICAO    string            `json:"dest_icao"`
	SampleCount int               `json:"sample_count"`
	Segments    []CorridorSegment `json:"segments"`
	UpdatedAt   int64             `json:"updated_at"`
}

// GetRouteCorridor fetches the learned corridor for an airport pair.
func (db *DB) GetRouteCorridor(origICAO, destICAO string) (*RouteCorridor, bool, error) {
	orig := strings.ToUpper(strings.TrimSpace(origICAO))
	dest := strings.ToUpper(strings.TrimSpace(destICAO))
	if orig == "" || dest == "" {
		return nil, false, nil
	}

	var sampleCount int
	var dataJSON string
	var updatedAt int64
	err := db.QueryRow(`
		SELECT sample_count, corridor_data_json, updated_at
		FROM route_corridors
		WHERE origin_icao = ? AND dest_icao = ?
	`, orig, dest).Scan(&sampleCount, &dataJSON, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var segments []CorridorSegment
	if err := json.Unmarshal([]byte(dataJSON), &segments); err != nil {
		return nil, false, err
	}

	return &RouteCorridor{
		OriginICAO:  orig,
		DestICAO:    dest,
		SampleCount: sampleCount,
		Segments:    segments,
		UpdatedAt:   updatedAt,
	}, true, nil
}

// SaveRouteCorridor saves or updates a learned flight corridor.
func (db *DB) SaveRouteCorridor(origICAO, destICAO string, sampleCount int, segments []CorridorSegment) error {
	orig := strings.ToUpper(strings.TrimSpace(origICAO))
	dest := strings.ToUpper(strings.TrimSpace(destICAO))
	dataJSON, err := json.Marshal(segments)
	if err != nil {
		return err
	}
	now := time.Now().Unix()

	_, err = db.Exec(`
		INSERT INTO route_corridors (origin_icao, dest_icao, sample_count, corridor_data_json, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(origin_icao, dest_icao) DO UPDATE SET
			sample_count = excluded.sample_count,
			corridor_data_json = excluded.corridor_data_json,
			updated_at = excluded.updated_at
	`, orig, dest, sampleCount, string(dataJSON), now)
	return err
}
