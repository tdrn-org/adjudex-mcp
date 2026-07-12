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
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-database"
)

type Position struct {
	ID          string  `db:"id"`
	PortfolioID string  `db:"portfolio_id"`
	Symbol      string  `db:"symbol"`
	Quantity    float64 `db:"quantity"`
	EntryPrice  float64 `db:"entry_price"`
	EntryDate   int64   `db:"entry_date"`
	Notes       string  `db:"notes"`
	CreatedAt   int64   `db:"created_at"`
	UpdatedAt   int64   `db:"updated_at"`
}

//go:embed position.insert.sql
var insertPositionSQL string

func InsertPosition(ctx context.Context, driver *database.Driver, portfolioID string, pos *domain.Position) (*Position, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	position := &Position{
		ID:          database.NewID(),
		PortfolioID: portfolioID,
		Symbol:      pos.Symbol,
		Quantity:    pos.Quantity,
		EntryPrice:  pos.EntryPrice,
		EntryDate:   database.Time2DB(pos.EntryDate),
		Notes:       pos.Notes,
		CreatedAt:   database.Time2DB(tx.Now()),
		UpdatedAt:   database.Time2DB(tx.Now()),
	}
	err = tx.ExecTx(txCtx, insertPositionSQL,
		position.ID,
		position.PortfolioID,
		position.Symbol,
		position.Quantity,
		position.EntryPrice,
		position.EntryDate,
		position.Notes,
		position.CreatedAt,
		position.UpdatedAt)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}
	return position, nil
}

//go:embed position.update_by_id.sql
var updatePositionByIDSQL string

func UpdatePosition(ctx context.Context, driver *database.Driver, portfolioID string, pos *domain.Position) (*Position, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	position := &Position{
		ID:          pos.ID,
		PortfolioID: portfolioID,
		Symbol:      pos.Symbol,
		Quantity:    pos.Quantity,
		EntryPrice:  pos.EntryPrice,
		EntryDate:   database.Time2DB(pos.EntryDate),
		Notes:       pos.Notes,
		UpdatedAt:   database.Time2DB(tx.Now()),
	}
	err = tx.ExecTx(txCtx, updatePositionByIDSQL,
		position.Symbol,
		position.Quantity,
		position.EntryPrice,
		position.EntryDate,
		position.Notes,
		position.UpdatedAt,
		position.ID,
	)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return position, nil
}

//go:embed position.select_by_portfolio_id.sql
var selectPositionByPortfolioIDSQL string

func SelectPositionsByPortfolioID(ctx context.Context, driver *database.Driver, portfolioID string) ([]*Position, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectPositionByPortfolioIDSQL, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]*Position, 0)
	for rows.Next() {
		position := &Position{
			PortfolioID: portfolioID,
		}
		err = database.Scan(rows, position)
		if err != nil {
			return nil, err
		}
		positions = append(positions, position)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return positions, nil
}

//go:embed position.delete_by_id.sql
var deletePositionByIDSQL string

func DeletePositionByID(ctx context.Context, driver *database.Driver, id string) error {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, deletePositionByIDSQL, id)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

//go:embed position.delete_by_portfolio_id.sql
var deletePositionByPortfolioIDSQL string

func DeletePositionsByPortfolioID(ctx context.Context, driver *database.Driver, portfolioID string) error {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, deletePositionByPortfolioIDSQL, portfolioID)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

//go:embed position.select_symbols.sql
var selectPositionSymbolsSQL string

func SelectPositionSymbols(ctx context.Context, driver *database.Driver) (map[string]time.Time, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectPositionSymbolsSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	symbolMap := make(map[string]time.Time, 0)
	for rows.Next() {
		var symbol string
		var timestamp *int64
		err = rows.Scan(&symbol, &timestamp)
		if err != nil {
			return nil, err
		}
		if timestamp != nil {
			symbolMap[symbol] = database.DB2Time(*timestamp)
		} else {
			symbolMap[symbol] = time.Time{}
		}
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return symbolMap, nil
}
