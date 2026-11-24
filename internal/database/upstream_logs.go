package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"contrail/internal/models"
)

// UpstreamFetchLog represents a telemetry provider ingestion attempt.
type UpstreamFetchLog struct {
	ID            int64  `json:"id"`
	Timestamp     int64  `json:"timestamp"`
	Provider      string `json:"provider"`
	Hub           string `json:"hub,omitempty"`
	Status        string `json:"status"` // "success", "rate_limited", "error"
	StatusCode    int    `json:"status_code"`
	AircraftCount int    `json:"aircraft_count"`
	DurationMs    int64  `json:"duration_ms"`
	ErrorMessage  string `json:"error_message,omitempty"`
	RepeatCount   int    `json:"repeat_count,omitempty"`
}

// LogUpstreamFetch records an upstream fetch event with optional regional hub.
// Consecutive 429 rate limit responses for the same hub are merged into one record with a repeat counter.
func (db *DB) LogUpstreamFetch(provider, hub, status string, statusCode, aircraftCount int, durationMs int64, errorMsg string) error {
	now := time.Now().Unix()

	if statusCode == 429 {
		var lastID int64
		var lastCode int
		var lastErr string
		var lastRepeat int
		err := db.QueryRow(`
			SELECT id, status_code, COALESCE(error_message, ''), COALESCE(repeat_count, 1)
			FROM upstream_fetch_logs
			WHERE provider = ? AND hub = ?
			ORDER BY id DESC LIMIT 1
		`, provider, hub).Scan(&lastID, &lastCode, &lastErr, &lastRepeat)

		if err == nil && lastCode == 429 {
			newRepeat := lastRepeat + 1
			baseMsg := errorMsg
			if idx := strings.Index(baseMsg, " (x"); idx != -1 {
				baseMsg = strings.TrimSpace(baseMsg[:idx])
			}
			updatedMsg := fmt.Sprintf("%s (x%d)", baseMsg, newRepeat)
			_, err = db.Exec(`
				UPDATE upstream_fetch_logs
				SET timestamp = ?, duration_ms = ?, error_message = ?, repeat_count = ?
				WHERE id = ?
			`, now, durationMs, updatedMsg, newRepeat, lastID)
			return err
		}
	}

	_, err := db.Exec(`
		INSERT INTO upstream_fetch_logs (timestamp, provider, hub, status, status_code, aircraft_count, duration_ms, error_message, repeat_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
	`, now, provider, hub, status, statusCode, aircraftCount, durationMs, errorMsg)
	if err != nil {
		return err
	}

	// Housekeep: retain up to last 2000 records to prevent growth
	_, _ = db.Exec("DELETE FROM upstream_fetch_logs WHERE id NOT IN (SELECT id FROM upstream_fetch_logs ORDER BY id DESC LIMIT 2000)")
	return nil
}

// GetUpstreamFetchLogs retrieves recent upstream ingestion log entries with optional filtering.
func (db *DB) GetUpstreamFetchLogs(limit int, providerFilter, hubFilter, statusFilter string) ([]*UpstreamFetchLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		SELECT id, timestamp, provider, COALESCE(hub, ''), status, status_code, aircraft_count, duration_ms, COALESCE(error_message, ''), COALESCE(repeat_count, 1)
		FROM upstream_fetch_logs
	`
	var whereClauses []string
	var args []interface{}

	if p := strings.TrimSpace(providerFilter); p != "" && p != "all" {
		whereClauses = append(whereClauses, "LOWER(provider) = LOWER(?)")
		args = append(args, p)
	}
	if h := strings.TrimSpace(hubFilter); h != "" {
		whereClauses = append(whereClauses, "LOWER(hub) LIKE LOWER(?)")
		args = append(args, "%"+h+"%")
	}
	if s := strings.TrimSpace(statusFilter); s != "" && s != "all" {
		whereClauses = append(whereClauses, "LOWER(status) = LOWER(?)")
		args = append(args, s)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY timestamp DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*UpstreamFetchLog
	for rows.Next() {
		var l UpstreamFetchLog
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.Provider, &l.Hub, &l.Status, &l.StatusCode, &l.AircraftCount, &l.DurationMs, &l.ErrorMessage, &l.RepeatCount); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	if logs == nil {
		logs = []*UpstreamFetchLog{}
	}
	return logs, nil
}

// SaveUpstreamState updates or inserts the persistent provider budget/cooldown state.
func (db *DB) SaveUpstreamState(state *models.UpstreamState) error {
	if state == nil {
		return nil
	}
	now := time.Now().Unix()
	state.UpdatedAt = now

	_, err := db.Exec(`
		INSERT INTO upstream_state (
			provider, status, last_success_at, next_allowed_at, remaining_credits,
			consecutive_failures, cooldown_until, last_fetched_count, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider) DO UPDATE SET
			status=excluded.status,
			last_success_at=excluded.last_success_at,
			next_allowed_at=excluded.next_allowed_at,
			remaining_credits=excluded.remaining_credits,
			consecutive_failures=excluded.consecutive_failures,
			cooldown_until=excluded.cooldown_until,
			last_fetched_count=excluded.last_fetched_count,
			updated_at=excluded.updated_at
	`, state.Provider, state.Status, state.LastSuccessAt, state.NextAllowedAt, state.RemainingCredits,
		state.ConsecutiveFailures, state.CooldownUntil, state.LastFetchedCount, state.UpdatedAt)
	return err
}

// GetUpstreamState retrieves the stored state of an upstream provider.
func (db *DB) GetUpstreamState(provider string) (*models.UpstreamState, error) {
	row := db.QueryRow(`
		SELECT provider, status, last_success_at, next_allowed_at, remaining_credits,
		       consecutive_failures, cooldown_until, last_fetched_count, updated_at
		FROM upstream_state
		WHERE LOWER(provider) = LOWER(?)
	`, strings.TrimSpace(provider))

	var s models.UpstreamState
	err := row.Scan(&s.Provider, &s.Status, &s.LastSuccessAt, &s.NextAllowedAt, &s.RemainingCredits,
		&s.ConsecutiveFailures, &s.CooldownUntil, &s.LastFetchedCount, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetAllUpstreamStates loads all provider states.
func (db *DB) GetAllUpstreamStates() ([]*models.UpstreamState, error) {
	rows, err := db.Query(`
		SELECT provider, status, last_success_at, next_allowed_at, remaining_credits,
		       consecutive_failures, cooldown_until, last_fetched_count, updated_at
		FROM upstream_state
		ORDER BY provider ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.UpstreamState
	for rows.Next() {
		var s models.UpstreamState
		if err := rows.Scan(&s.Provider, &s.Status, &s.LastSuccessAt, &s.NextAllowedAt, &s.RemainingCredits,
			&s.ConsecutiveFailures, &s.CooldownUntil, &s.LastFetchedCount, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}
	return list, nil
}
