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

package model

import (
	"context"
	_ "embed"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-finance"
)

type Position struct {
	ID          string                  `db:"id"`
	PortfolioID string                  `db:"portfolio_id"`
	Listing     finance.ProviderListing `db:"listing"`
	Currency    finance.Currency        `db:"currency"`
	Quantity    float64                 `db:"quantity"`
	EntryPrice  float64                 `db:"entry_price"`
	EntryDate   int64                   `db:"entry_date"`
	Notes       string                  `db:"notes"`
	CreatedAt   int64                   `db:"created_at"`
	UpdatedAt   int64                   `db:"updated_at"`
}

//go:embed position.insert.sql
var insertPositionSQL string

func InsertPosition(ctx context.Context, tx *database.Tx, portfolioID string, position *domain.Position) (*Position, error) {
	p := &Position{
		ID:          database.NewID(),
		PortfolioID: portfolioID,
		Listing:     position.Listing,
		Currency:    position.Currency,
		Quantity:    position.Quantity,
		EntryPrice:  position.EntryPrice,
		EntryDate:   database.Time2DB(position.EntryDate),
		Notes:       position.Notes,
		CreatedAt:   database.Time2DB(tx.Now()),
		UpdatedAt:   database.Time2DB(tx.Now()),
	}
	err := tx.ExecTx(ctx, insertPositionSQL,
		p.ID,
		p.PortfolioID,
		p.Listing.ID(),
		p.Currency,
		p.Quantity,
		p.EntryPrice,
		p.EntryDate,
		p.Notes,
		p.CreatedAt,
		p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

//go:embed position.select_by_portfolio_id.sql
var selectPositionByPortfolioIDSQL string

func SelectPositionsByPortfolioID(ctx context.Context, tx *database.Tx, portfolioID string) ([]*Position, error) {
	rows, err := tx.QueryTx(ctx, selectPositionByPortfolioIDSQL,
		portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ps := make([]*Position, 0)
	for rows.Next() {
		p := &Position{
			PortfolioID: portfolioID,
		}
		err = database.Scan(rows, p)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

//go:embed position.update_by_id.sql
var updatePositionByIDSQL string

func UpdatePosition(ctx context.Context, tx *database.Tx, portfolioID string, position *domain.Position) (*Position, error) {
	p := &Position{
		ID:          position.ID,
		PortfolioID: portfolioID,
		Listing:     position.Listing,
		Currency:    position.Currency,
		Quantity:    position.Quantity,
		EntryPrice:  position.EntryPrice,
		EntryDate:   database.Time2DB(position.EntryDate),
		Notes:       position.Notes,
		UpdatedAt:   database.Time2DB(tx.Now()),
	}
	err := tx.ExecTx(ctx, updatePositionByIDSQL,
		p.Listing.ID(),
		p.Currency,
		p.Quantity,
		p.EntryPrice,
		p.EntryDate,
		p.Notes,
		p.UpdatedAt,
		p.ID,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

//go:embed position.delete_by_id.sql
var deletePositionByIDSQL string

func DeletePositionByID(ctx context.Context, tx *database.Tx, id string) error {
	return tx.ExecTx(ctx, deletePositionByIDSQL, id)
}

//go:embed position.delete_by_portfolio_id.sql
var deletePositionByPortfolioIDSQL string

func DeletePositionsByPortfolioID(ctx context.Context, tx *database.Tx, portfolioID string) error {
	return tx.ExecTx(ctx, deletePositionByPortfolioIDSQL, portfolioID)
}
