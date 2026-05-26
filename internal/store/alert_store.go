/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package store

import (
	"context"
	"fmt"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-database"
)

// CreateAlert persists a new alert with a generated ID in armed state.
func (s *Store) CreateAlert(ctx context.Context, a *domain.Alert) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("creating alert %q: begin tx: %w", a.Name, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	a.ID = database.NewID()
	now := database.Now()
	a.State = domain.AlertStateArmed
	a.CreatedAt = database.DB2Time(now)
	a.UpdatedAt = a.CreatedAt

	// Encode optional indicator as JSON strings
	var indicatorType, indicatorPeriod string
	if a.Indicator != nil {
		indicatorType = string(a.Indicator.Type)
		indicatorPeriod = itoa(a.Indicator.Period)
	}

	err = tx.ExecTx(ctx,
		"INSERT INTO alerts (id, name, symbol, condition, threshold, indicator_type, indicator_period, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		a.ID, a.Name, a.Symbol, string(a.Condition), a.Threshold, indicatorType, indicatorPeriod, string(a.State), now, now,
	)
	if err != nil {
		return fmt.Errorf("creating alert %q: insert: %w", a.Name, err)
	}
	return tx.CommitTx(ctx)
}

// GetAlert retrieves an alert by ID.
func (s *Store) GetAlert(ctx context.Context, id string) (*domain.Alert, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting alert %q: begin tx: %w", id, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	row, err := tx.QueryRowTx(ctx,
		"SELECT id, name, symbol, condition, threshold, indicator_type, indicator_period, state, triggered_at, message, created_at, updated_at FROM alerts WHERE id = ?", id,
	)
	if err != nil {
		return nil, fmt.Errorf("getting alert %q: query: %w", id, err)
	}

	a, err := scanAlert(row)
	if err != nil {
		if database.NoRows(err) {
			return nil, fmt.Errorf("alert %q: %w", id, err)
		}
		return nil, fmt.Errorf("getting alert %q: %w", id, err)
	}
	return a, tx.CommitTx(ctx)
}

// ListAlerts returns all alerts, optionally filtered by symbol.
func (s *Store) ListAlerts(ctx context.Context, symbol string) ([]domain.Alert, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing alerts: begin tx: %w", err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	var rows interface {
		Close() error
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	var queryErr error
	if symbol == "" {
		rows, queryErr = tx.QueryTx(ctx, "SELECT id, name, symbol, condition, threshold, indicator_type, indicator_period, state, triggered_at, message, created_at, updated_at FROM alerts ORDER BY created_at DESC")
	} else {
		rows, queryErr = tx.QueryTx(ctx, "SELECT id, name, symbol, condition, threshold, indicator_type, indicator_period, state, triggered_at, message, created_at, updated_at FROM alerts WHERE symbol = ? ORDER BY created_at DESC", symbol)
	}
	if queryErr != nil {
		return nil, fmt.Errorf("listing alerts: query: %w", queryErr)
	}
	defer rows.Close()

	result := make([]domain.Alert, 0)
	for rows.Next() {
		a, err := scanAlertRow(rows)
		if err != nil {
			return nil, fmt.Errorf("listing alerts: scan: %w", err)
		}
		result = append(result, *a)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("listing alerts: rows iteration: %w", err)
	}

	return result, tx.CommitTx(ctx)
}

// ListArmedAlerts returns all alerts currently in armed state.
func (s *Store) ListArmedAlerts(ctx context.Context) ([]domain.Alert, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing armed alerts: begin tx: %w", err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	rows, err := tx.QueryTx(ctx,
		"SELECT id, name, symbol, condition, threshold, indicator_type, indicator_period, state, triggered_at, message, created_at, updated_at FROM alerts WHERE state = ? ORDER BY created_at DESC",
		string(domain.AlertStateArmed),
	)
	if err != nil {
		return nil, fmt.Errorf("listing armed alerts: query: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Alert, 0)
	for rows.Next() {
		a, err := scanAlertRow(rows)
		if err != nil {
			return nil, fmt.Errorf("listing armed alerts: scan: %w", err)
		}
		result = append(result, *a)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("listing armed alerts: rows iteration: %w", err)
	}

	return result, tx.CommitTx(ctx)
}

// UpdateAlert persists changes to an existing alert (state transitions etc.).
func (s *Store) UpdateAlert(ctx context.Context, a *domain.Alert) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("updating alert %q: begin tx: %w", a.ID, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	now := database.Now()
	a.UpdatedAt = database.DB2Time(now)

	var triggeredAt *int64
	if a.TriggeredAt != nil {
		ts := database.Time2DB(*a.TriggeredAt)
		triggeredAt = &ts
	}

	err = tx.ExecTx(ctx,
		"UPDATE alerts SET state = ?, triggered_at = ?, message = ?, updated_at = ? WHERE id = ?",
		string(a.State), triggeredAt, a.Message, now, a.ID,
	)
	if err != nil {
		return fmt.Errorf("updating alert %q: update: %w", a.ID, err)
	}
	return tx.CommitTx(ctx)
}

// DeleteAlert removes an alert.
func (s *Store) DeleteAlert(ctx context.Context, id string) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("deleting alert %q: begin tx: %w", id, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	err = tx.ExecTx(ctx, "DELETE FROM alerts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting alert %q: delete: %w", id, err)
	}
	return tx.CommitTx(ctx)
}

// scanAlert scans an alert from a sql.Row (single row).
func scanAlert(scanner interface{ Scan(dest ...any) error }) (*domain.Alert, error) {
	var a domain.Alert
	var indicatorType, indicatorPeriod string
	var state string
	var triggeredAt *int64
	var createdAt, updatedAt int64

	err := scanner.Scan(&a.ID, &a.Name, &a.Symbol, &a.Condition, &a.Threshold,
		&indicatorType, &indicatorPeriod, &state, &triggeredAt, &a.Message,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	a.State = domain.AlertState(state)
	a.CreatedAt = database.DB2Time(createdAt)
	a.UpdatedAt = database.DB2Time(updatedAt)

	if triggeredAt != nil {
		t := database.DB2Time(*triggeredAt)
		a.TriggeredAt = &t
	}

	// Reconstruct indicator spec if present
	if indicatorType != "" || indicatorPeriod != "" {
		a.Indicator = &domain.IndicatorSpec{
			Type:   domain.IndicatorType(indicatorType),
			Period: atoi(indicatorPeriod),
		}
	}

	return &a, nil
}

// scanAlertRow scans an alert from a sql.Rows iterator (multi-row).
func scanAlertRow(scanner interface {
	Scan(dest ...any) error
	Err() error
}) (*domain.Alert, error) {
	return scanAlert(scanner)
}

// atoi converts a decimal string to int with error sentinel (panics for invalid input).
func atoi(s string) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0
	}
	return n
}

// itoa converts an int to a decimal string.
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
