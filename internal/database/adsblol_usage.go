package database

import "time"

// RecordADSBLolUsage records an HTTP request made to api.adsb.lol.
func (db *DB) RecordADSBLolUsage(timestamp int64, endpoint string, statusCode int) error {
	_, err := db.Exec(`
		INSERT INTO adsblol_usage (timestamp, endpoint, status_code, count)
		VALUES (?, ?, ?, 1)
	`, timestamp, endpoint, statusCode)
	return err
}

// GetADSBLolUsageInPast24Hours returns total requests made to adsb.lol in the rolling 24-hour window.
func (db *DB) GetADSBLolUsageInPast24Hours(nowUnix int64) (int, error) {
	windowStart := nowUnix - 86400
	var total int
	err := db.QueryRow(`
		SELECT COALESCE(SUM(count), 0)
		FROM adsblol_usage
		WHERE timestamp >= ?
	`, windowStart).Scan(&total)
	return total, err
}

// GetADSBLolUsageToday returns requests made during the current 24-hour UTC day, and the next reset timestamp.
func (db *DB) GetADSBLolUsageToday(nowUnix int64) (used int, windowResetAt int64, err error) {
	now := time.Unix(nowUnix, 0).UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC).Unix()

	err = db.QueryRow(`
		SELECT COALESCE(SUM(count), 0)
		FROM adsblol_usage
		WHERE timestamp >= ?
	`, startOfDay).Scan(&used)
	return used, nextMidnight, err
}
