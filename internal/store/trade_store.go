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

// RecordTrade persists a new trade with a generated ID.
func (s *Store) RecordTrade(ctx context.Context, t *domain.Trade) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("recording trade for %q: begin tx: %w", t.Symbol, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	t.ID = database.NewID()

	err = tx.ExecTx(ctx,
		"INSERT INTO trades (id, strategy_id, symbol, direction, quantity, price, executed_at, status, pnl, notes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.StrategyID, t.Symbol, string(t.Direction), t.Quantity, t.Price, database.Time2DB(t.ExecutedAt), string(t.Status), t.PnL, t.Notes,
	)
	if err != nil {
		return fmt.Errorf("recording trade for %q: insert: %w", t.Symbol, err)
	}
	return tx.CommitTx(ctx)
}

// GetTrade retrieves a trade by ID.
func (s *Store) GetTrade(ctx context.Context, id string) (*domain.Trade, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting trade %q: begin tx: %w", id, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	row, err := tx.QueryRowTx(ctx,
		"SELECT id, strategy_id, symbol, direction, quantity, price, executed_at, status, pnl, notes FROM trades WHERE id = ?", id,
	)
	if err != nil {
		return nil, fmt.Errorf("getting trade %q: query: %w", id, err)
	}

	var t domain.Trade
	var direction, status string
	var executedAt int64
	err = row.Scan(&t.ID, &t.StrategyID, &t.Symbol, &direction, &t.Quantity, &t.Price, &executedAt, &status, &t.PnL, &t.Notes)
	if err != nil {
		if database.NoRows(err) {
			return nil, fmt.Errorf("trade %q: %w", id, err)
		}
		return nil, fmt.Errorf("getting trade %q: scan: %w", id, err)
	}
	t.Direction = domain.TradeDirection(direction)
	t.Status = domain.TradeStatus(status)
	t.ExecutedAt = database.DB2Time(executedAt)

	return &t, tx.CommitTx(ctx)
}

// ListTrades returns all trades, optionally filtered by symbol.
func (s *Store) ListTrades(ctx context.Context, symbol string) ([]domain.Trade, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing trades: begin tx: %w", err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	return s.queryTrades(ctx, tx, "SELECT id, strategy_id, symbol, direction, quantity, price, executed_at, status, pnl, notes FROM trades WHERE symbol = ? ORDER BY executed_at DESC", symbol)
}

// ListTradesByStrategy returns all trades for a specific strategy.
func (s *Store) ListTradesByStrategy(ctx context.Context, strategyID string) ([]domain.Trade, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing trades by strategy %q: begin tx: %w", strategyID, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	return s.queryTrades(ctx, tx, "SELECT id, strategy_id, symbol, direction, quantity, price, executed_at, status, pnl, notes FROM trades WHERE strategy_id = ? ORDER BY executed_at DESC", strategyID)
}

// queryTrades executes a parameterized trade query and returns the results.
func (s *Store) queryTrades(ctx context.Context, tx *database.Tx, query string, arg string) ([]domain.Trade, error) {
	rows, err := tx.QueryTx(ctx, query, arg)
	if err != nil {
		return nil, fmt.Errorf("listing trades: query: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Trade, 0)
	for rows.Next() {
		var t domain.Trade
		var direction, status string
		var executedAt int64
		if err = rows.Scan(&t.ID, &t.StrategyID, &t.Symbol, &direction, &t.Quantity, &t.Price, &executedAt, &status, &t.PnL, &t.Notes); err != nil {
			return nil, fmt.Errorf("listing trades: scan: %w", err)
		}
		t.Direction = domain.TradeDirection(direction)
		t.Status = domain.TradeStatus(status)
		t.ExecutedAt = database.DB2Time(executedAt)
		result = append(result, t)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("listing trades: rows iteration: %w", err)
	}

	return result, tx.CommitTx(ctx)
}
