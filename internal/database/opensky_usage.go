package database

import "strings"

// RecordOpenSkyUsage logs a single OpenSky request with credits consumed, status code, reported remaining quota,
// and the query type ('global' for worldwide snapshot, 'regional' for bounding box calibration).
func (db *DB) RecordOpenSkyUsage(timestamp int64, queryType string, creditsUsed int, statusCode int, remaining int) error {
	qt := strings.ToLower(strings.TrimSpace(queryType))
	if qt == "" {
		qt = "global"
	}
	_, err := db.Exec(`
		INSERT INTO opensky_usage (timestamp, query_type, credits_used, status_code, rate_limit_remaining)
		VALUES (?, ?, ?, ?, ?)
	`, timestamp, qt, creditsUsed, statusCode, remaining)
	return err
}

// GetOpenSkyUsageInPast24Hours sums all credits deducted by OpenSky in the rolling 24-hour window [nowUnix - 86400, nowUnix].
func (db *DB) GetOpenSkyUsageInPast24Hours(nowUnix int64) (int, error) {
	windowStart := nowUnix - 86400
	var totalUsed int
	err := db.QueryRow(`
		SELECT COALESCE(SUM(credits_used), 0)
		FROM opensky_usage
		WHERE timestamp >= ?
	`, windowStart).Scan(&totalUsed)
	if err != nil {
		return 0, err
	}
	return totalUsed, nil
}

// GetOpenSkyUsageInPast24HoursByType sums credits deducted by OpenSky query type ('global' vs 'regional') in rolling 24h.
func (db *DB) GetOpenSkyUsageInPast24HoursByType(nowUnix int64, queryType string) (int, error) {
	windowStart := nowUnix - 86400
	qt := strings.ToLower(strings.TrimSpace(queryType))
	var totalUsed int
	err := db.QueryRow(`
		SELECT COALESCE(SUM(credits_used), 0)
		FROM opensky_usage
		WHERE timestamp >= ? AND query_type = ?
	`, windowStart, qt).Scan(&totalUsed)
	if err != nil {
		return 0, err
	}
	return totalUsed, nil
}
