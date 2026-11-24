package database

import (
	"math"
	"sort"
)

// RecordExtrapolationAccuracy persists the displacement error between an extrapolated flight position and a newly arrived real sensor observation.
func (db *DB) RecordExtrapolationAccuracy(hex string, timestamp int64, extrapLat, extrapLon, obsLat, obsLon, errorKm float64, ageSeconds int) error {
	_, err := db.Exec(`
		INSERT INTO extrapolation_accuracy_logs (hex, timestamp, extrapolated_lat, extrapolated_lon, observed_lat, observed_lon, error_km, age_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, hex, timestamp, extrapLat, extrapLon, obsLat, obsLon, errorKm, ageSeconds)

	// Clean up old accuracy records > 7 days to keep database compact
	_, _ = db.Exec("DELETE FROM extrapolation_accuracy_logs WHERE timestamp < ?", timestamp-7*86400)
	return err
}

// GetExtrapolationAccuracyStats computes the mean and P95 displacement error (in km) over the specified past hours.
func (db *DB) GetExtrapolationAccuracyStats(pastHours int) (meanKm, p95Km float64, totalSamples int, err error) {
	if pastHours <= 0 {
		pastHours = 24
	}
	cutoff := int64(pastHours * 3600)
	rows, err := db.Query(`
		SELECT error_km
		FROM extrapolation_accuracy_logs
		WHERE timestamp >= (strftime('%s', 'now') - ?)
		ORDER BY error_km ASC
	`, cutoff)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	var errors []float64
	var sum float64
	for rows.Next() {
		var val float64
		if err := rows.Scan(&val); err == nil {
			errors = append(errors, val)
			sum += val
		}
	}

	totalSamples = len(errors)
	if totalSamples == 0 {
		return 0.0, 0.0, 0, nil
	}

	meanKm = math.Round((sum/float64(totalSamples))*100.0) / 100.0

	// Compute 95th percentile
	sort.Float64s(errors)
	p95Idx := int(float64(totalSamples) * 0.95)
	if p95Idx >= totalSamples {
		p95Idx = totalSamples - 1
	}
	p95Km = math.Round(errors[p95Idx]*100.0) / 100.0

	return meanKm, p95Km, totalSamples, nil
}
