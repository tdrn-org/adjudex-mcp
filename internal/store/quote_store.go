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
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-database"
)

// SaveQuote persists a single quote record.
func (s *Store) SaveQuote(ctx context.Context, q *domain.Quote) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("saving quote for %q: begin tx: %w", q.Symbol, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	err = tx.ExecTx(ctx,
		"INSERT INTO quotes (symbol, timestamp, open, high, low, close, volume, source) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		q.Symbol, database.Time2DB(q.Timestamp), q.Open, q.High, q.Low, q.Close, q.Volume, q.Source,
	)
	if err != nil {
		return fmt.Errorf("saving quote for %q: insert: %w", q.Symbol, err)
	}
	return tx.CommitTx(ctx)
}

// SaveQuotes persists multiple quotes in batch (single transaction).
func (s *Store) SaveQuotes(ctx context.Context, quotes []domain.Quote) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("saving %d quotes: begin tx: %w", len(quotes), err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	for i, q := range quotes {
		err = tx.ExecTx(ctx,
			"INSERT INTO quotes (symbol, timestamp, open, high, low, close, volume, source) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			q.Symbol, database.Time2DB(q.Timestamp), q.Open, q.High, q.Low, q.Close, q.Volume, q.Source,
		)
		if err != nil {
			return fmt.Errorf("saving quote %d/%d for %q: insert: %w", i+1, len(quotes), q.Symbol, err)
		}
	}
	return tx.CommitTx(ctx)
}

// GetQuotes returns quotes for a symbol within a date range.
func (s *Store) GetQuotes(ctx context.Context, symbol string, from, to time.Time) ([]domain.Quote, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting quotes for %q: begin tx: %w", symbol, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	rows, err := tx.QueryTx(ctx,
		"SELECT symbol, timestamp, open, high, low, close, volume, source FROM quotes WHERE symbol = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp ASC",
		symbol, database.Time2DB(from), database.Time2DB(to),
	)
	if err != nil {
		return nil, fmt.Errorf("getting quotes for %q: query: %w", symbol, err)
	}
	defer rows.Close()

	result := make([]domain.Quote, 0)
	for rows.Next() {
		var q domain.Quote
		var ts int64
		if err = rows.Scan(&q.Symbol, &ts, &q.Open, &q.High, &q.Low, &q.Close, &q.Volume, &q.Source); err != nil {
			return nil, fmt.Errorf("getting quotes for %q: scan: %w", symbol, err)
		}
		q.Timestamp = database.DB2Time(ts)
		result = append(result, q)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("getting quotes for %q: rows iteration: %w", symbol, err)
	}

	return result, tx.CommitTx(ctx)
}

// GetLatestQuote returns the most recent quote for a symbol.
func (s *Store) GetLatestQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting latest quote for %q: begin tx: %w", symbol, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	row, err := tx.QueryRowTx(ctx,
		"SELECT symbol, timestamp, open, high, low, close, volume, source FROM quotes WHERE symbol = ? ORDER BY timestamp DESC LIMIT 1",
		symbol,
	)
	if err != nil {
		return nil, fmt.Errorf("getting latest quote for %q: query: %w", symbol, err)
	}
	var q domain.Quote
	var ts int64
	err = row.Scan(&q.Symbol, &ts, &q.Open, &q.High, &q.Low, &q.Close, &q.Volume, &q.Source)
	if err != nil {
		if database.NoRows(err) {
			return nil, fmt.Errorf("no quote found for %q: %w", symbol, err)
		}
		return nil, fmt.Errorf("getting latest quote for %q: scan: %w", symbol, err)
	}
	q.Timestamp = database.DB2Time(ts)
	return &q, tx.CommitTx(ctx)
}
