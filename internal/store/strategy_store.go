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

// SaveStrategy persists a strategy (create or update) with a generated ID if new.
func (s *Store) SaveStrategy(ctx context.Context, st *domain.Strategy) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("saving strategy %q: begin tx: %w", st.Name, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	if st.ID == "" {
		st.ID = database.NewID()
	}
	now := database.Now()
	st.UpdatedAt = database.DB2Time(now)
	if st.CreatedAt.IsZero() {
		st.CreatedAt = st.UpdatedAt
	}

	err = tx.ExecTx(ctx,
		`INSERT INTO strategies (id, name, description, rsi_period, rsi_threshold, sma_period, sma_trigger, max_position, stop_loss, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 name = excluded.name, description = excluded.description,
		 rsi_period = excluded.rsi_period, rsi_threshold = excluded.rsi_threshold,
		 sma_period = excluded.sma_period, sma_trigger = excluded.sma_trigger,
		 max_position = excluded.max_position, stop_loss = excluded.stop_loss,
		 updated_at = excluded.updated_at`,
		st.ID, st.Name, st.Description,
		st.Parameters.RSIPeriod, st.Parameters.RSIThreshold,
		st.Parameters.SMAPeriod, st.Parameters.SMATrigger,
		st.Parameters.MaxPosition, st.Parameters.StopLoss,
		database.Time2DB(st.CreatedAt), now,
	)
	if err != nil {
		return fmt.Errorf("saving strategy %q: insert/update: %w", st.Name, err)
	}
	return tx.CommitTx(ctx)
}

// GetStrategy retrieves a strategy by ID.
func (s *Store) GetStrategy(ctx context.Context, id string) (*domain.Strategy, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting strategy %q: begin tx: %w", id, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	row, err := tx.QueryRowTx(ctx,
		"SELECT id, name, description, rsi_period, rsi_threshold, sma_period, sma_trigger, max_position, stop_loss, created_at, updated_at FROM strategies WHERE id = ?", id,
	)
	if err != nil {
		return nil, fmt.Errorf("getting strategy %q: query: %w", id, err)
	}

	var st domain.Strategy
	var createdAt, updatedAt int64
	err = row.Scan(&st.ID, &st.Name, &st.Description,
		&st.Parameters.RSIPeriod, &st.Parameters.RSIThreshold,
		&st.Parameters.SMAPeriod, &st.Parameters.SMATrigger,
		&st.Parameters.MaxPosition, &st.Parameters.StopLoss,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if database.NoRows(err) {
			return nil, fmt.Errorf("strategy %q: %w", id, err)
		}
		return nil, fmt.Errorf("getting strategy %q: scan: %w", id, err)
	}
	st.CreatedAt = database.DB2Time(createdAt)
	st.UpdatedAt = database.DB2Time(updatedAt)

	return &st, tx.CommitTx(ctx)
}

// ListStrategies returns all strategies.
func (s *Store) ListStrategies(ctx context.Context) ([]domain.Strategy, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing strategies: begin tx: %w", err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	rows, err := tx.QueryTx(ctx,
		"SELECT id, name, description, rsi_period, rsi_threshold, sma_period, sma_trigger, max_position, stop_loss, created_at, updated_at FROM strategies ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("listing strategies: query: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Strategy, 0)
	for rows.Next() {
		var st domain.Strategy
		var createdAt, updatedAt int64
		if err = rows.Scan(&st.ID, &st.Name, &st.Description,
			&st.Parameters.RSIPeriod, &st.Parameters.RSIThreshold,
			&st.Parameters.SMAPeriod, &st.Parameters.SMATrigger,
			&st.Parameters.MaxPosition, &st.Parameters.StopLoss,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("listing strategies: scan: %w", err)
		}
		st.CreatedAt = database.DB2Time(createdAt)
		st.UpdatedAt = database.DB2Time(updatedAt)
		result = append(result, st)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("listing strategies: rows iteration: %w", err)
	}

	return result, tx.CommitTx(ctx)
}

// DeleteStrategy removes a strategy.
func (s *Store) DeleteStrategy(ctx context.Context, id string) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("deleting strategy %q: begin tx: %w", id, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	err = tx.ExecTx(ctx, "DELETE FROM strategies WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting strategy %q: delete: %w", id, err)
	}
	return tx.CommitTx(ctx)
}
