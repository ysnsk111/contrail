package database

import (
	"context"
	"log"
	"time"
)

// StartRetentionCleaner starts a periodic worker that purges records older than retentionDays (90 days).
func (db *DB) StartRetentionCleaner(ctx context.Context, retentionDays int, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				db.CleanOldSnapshots(retentionDays)
				// Also clean up aircraft in live_flights that haven't been seen for > 30 minutes
				removed, err := db.RemoveStaleLiveFlights(30 * time.Minute)
				if err == nil && removed > 0 {
					log.Printf("[Cleaner] Pruned %d stale live aircraft (not seen for >30m)", removed)
				}
			}
		}
	}()
}

// CleanOldSnapshots removes snapshot history older than retentionDays.
func (db *DB) CleanOldSnapshots(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()

	res, err := db.Exec("DELETE FROM flight_snapshots WHERE timestamp < ?", cutoff)
	if err != nil {
		log.Printf("[Cleaner] Failed to delete expired snapshots: %v", err)
		return 0, err
	}

	rows, _ := res.RowsAffected()
	if rows > 0 {
		log.Printf("[Cleaner] Purged %d historical flight snapshots older than %d days", rows, retentionDays)
	}
	return rows, nil
}
