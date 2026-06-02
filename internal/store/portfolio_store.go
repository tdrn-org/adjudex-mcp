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

// Store implements all domain repository interfaces using SQLite.
type Store struct {
	db *database.Driver
}

// NewStore creates a new Store backed by the given database driver.
func NewStore(db *database.Driver) *Store {
	return &Store{db: db}
}

// Ensure Store satisfies all domain interfaces at compile time.
var (
	_ domain.PortfolioStore = (*Store)(nil)
	_ domain.QuoteStore     = (*Store)(nil)
	_ domain.AlertStore     = (*Store)(nil)
	_ domain.TradeStore     = (*Store)(nil)
	_ domain.StrategyStore  = (*Store)(nil)
)

// CreatePortfolio creates a new portfolio with a generated ID.
func (s *Store) CreatePortfolio(ctx context.Context, p *domain.Portfolio) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("creating portfolio %q: begin tx: %w", p.Name, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	p.ID = database.NewID()
	now := database.Now()
	p.CreatedAt = database.DB2Time(now)
	p.UpdatedAt = p.CreatedAt

	err = tx.ExecTx(ctx,
		"INSERT INTO portfolios (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Description, now, now,
	)
	if err != nil {
		return fmt.Errorf("creating portfolio %q: insert: %w", p.Name, err)
	}
	return tx.CommitTx(ctx)
}

// GetPortfolio retrieves a portfolio by ID, including its positions.
func (s *Store) GetPortfolio(ctx context.Context, id string) (*domain.Portfolio, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting portfolio %q: begin tx: %w", id, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	row, err := tx.QueryRowTx(ctx, "SELECT id, name, description, created_at, updated_at FROM portfolios WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("getting portfolio %q: query: %w", id, err)
	}
	var p domain.Portfolio
	var createdAt, updatedAt int64
	err = row.Scan(&p.ID, &p.Name, &p.Description, &createdAt, &updatedAt)
	if err != nil {
		if database.NoRows(err) {
			return nil, fmt.Errorf("portfolio %q: %w", id, err)
		}
		return nil, fmt.Errorf("getting portfolio %q: scan: %w", id, err)
	}
	p.CreatedAt = database.DB2Time(createdAt)
	p.UpdatedAt = database.DB2Time(updatedAt)

	// Load positions for this portfolio.
	rows, err := tx.QueryTx(ctx, "SELECT id, symbol, quantity, entry_price, entry_date, notes, created_at, updated_at FROM positions WHERE portfolio_id = ? ORDER BY created_at", id)
	if err != nil {
		return nil, fmt.Errorf("getting portfolio %q: query positions: %w", id, err)
	}
	defer rows.Close()

	p.Positions = make([]domain.Position, 0)
	for rows.Next() {
		var pos domain.Position
		var entryDate, posCreatedAt, posUpdatedAt int64
		err = rows.Scan(&pos.ID, &pos.Symbol, &pos.Quantity, &pos.EntryPrice, &entryDate, &pos.Notes, &posCreatedAt, &posUpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("getting portfolio %q: scan position: %w", id, err)
		}
		pos.EntryDate = database.DB2Time(entryDate)
		pos.CreatedAt = database.DB2Time(posCreatedAt)
		pos.UpdatedAt = database.DB2Time(posUpdatedAt)
		p.Positions = append(p.Positions, pos)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("getting portfolio %q: rows iteration: %w", id, err)
	}

	return &p, tx.CommitTx(ctx)
}

// ListPortfolios returns all portfolios (without positions for performance).
func (s *Store) ListPortfolios(ctx context.Context) ([]domain.Portfolio, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing portfolios: begin tx: %w", err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	rows, err := tx.QueryTx(ctx, "SELECT id, name, description, created_at, updated_at FROM portfolios ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("listing portfolios: query: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Portfolio, 0)
	for rows.Next() {
		var p domain.Portfolio
		var createdAt, updatedAt int64
		if err = rows.Scan(&p.ID, &p.Name, &p.Description, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("listing portfolios: scan: %w", err)
		}
		p.CreatedAt = database.DB2Time(createdAt)
		p.UpdatedAt = database.DB2Time(updatedAt)
		p.Positions = nil // not loaded in list mode
		result = append(result, p)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("listing portfolios: rows iteration: %w", err)
	}

	return result, tx.CommitTx(ctx)
}

// DeletePortfolio removes a portfolio and its positions (cascading).
func (s *Store) DeletePortfolio(ctx context.Context, id string) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("deleting portfolio %q: begin tx: %w", id, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	err = tx.ExecTx(ctx, "DELETE FROM positions WHERE portfolio_id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting portfolio %q: delete positions: %w", id, err)
	}
	err = tx.ExecTx(ctx, "DELETE FROM portfolios WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting portfolio %q: delete portfolio: %w", id, err)
	}
	return tx.CommitTx(ctx)
}

// AddPosition adds a new position to a portfolio.
func (s *Store) AddPosition(ctx context.Context, portfolioID string, pos *domain.Position) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("adding position to portfolio %q: begin tx: %w", portfolioID, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	pos.ID = database.NewID()
	now := database.Now()
	pos.CreatedAt = database.DB2Time(now)
	pos.UpdatedAt = pos.CreatedAt

	err = tx.ExecTx(ctx,
		"INSERT INTO positions (id, portfolio_id, symbol, quantity, entry_price, entry_date, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		pos.ID, portfolioID, pos.Symbol, pos.Quantity, pos.EntryPrice, database.Time2DB(pos.EntryDate), pos.Notes, now, now,
	)
	if err != nil {
		return fmt.Errorf("adding position to portfolio %q: insert: %w", portfolioID, err)
	}
	return tx.CommitTx(ctx)
}

// RemovePosition removes a position from a portfolio.
func (s *Store) RemovePosition(ctx context.Context, portfolioID string, positionID string) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("removing position %q from portfolio %q: begin tx: %w", positionID, portfolioID, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	err = tx.ExecTx(ctx, "DELETE FROM positions WHERE id = ? AND portfolio_id = ?", positionID, portfolioID)
	if err != nil {
		return fmt.Errorf("removing position %q from portfolio %q: delete: %w", positionID, portfolioID, err)
	}
	return tx.CommitTx(ctx)
}

// UpdatePosition updates an existing position's modifiable fields.
func (s *Store) UpdatePosition(ctx context.Context, portfolioID string, pos *domain.Position) error {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("updating position %q: begin tx: %w", pos.ID, err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	now := database.Now()
	pos.UpdatedAt = database.DB2Time(now)

	err = tx.ExecTx(ctx,
		"UPDATE positions SET symbol = ?, quantity = ?, entry_price = ?, entry_date = ?, notes = ?, updated_at = ? WHERE id = ? AND portfolio_id = ?",
		pos.Symbol, pos.Quantity, pos.EntryPrice, database.Time2DB(pos.EntryDate), pos.Notes, now, pos.ID, portfolioID,
	)
	if err != nil {
		return fmt.Errorf("updating position %q: update: %w", pos.ID, err)
	}
	return tx.CommitTx(ctx)
}

// ListSymbols returns all distinct symbols across all portfolios and positions.
func (s *Store) ListSymbols(ctx context.Context) ([]string, error) {
	_, tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing symbols: begin tx: %w", err)
	}
	defer tx.RollbackUncommitedTx(ctx)

	rows, err := tx.QueryTx(ctx, "SELECT DISTINCT symbol FROM positions ORDER BY symbol")
	if err != nil {
		return nil, fmt.Errorf("listing symbols: query: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var sym string
		if err := rows.Scan(&sym); err != nil {
			return nil, fmt.Errorf("listing symbols: scan: %w", err)
		}
		symbols = append(symbols, sym)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("listing symbols: rows iteration: %w", err)
	}
	return symbols, tx.CommitTx(ctx)
}
