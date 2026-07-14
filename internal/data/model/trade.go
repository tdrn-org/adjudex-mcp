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
)

type Trade struct {
	ID         string  `db:"id"`
	StrategyID string  `db:"strategy_id"`
	Symbol     string  `db:"symbol"`
	Currency   string  `db:"currency"`
	Direction  string  `db:"direction"`
	Quantity   float64 `db:"quantity"`
	Price      float64 `db:"price"`
	ExecutedAt int64   `db:"executed_at"`
	Status     string  `db:"status"`
	PnL        float64 `db:"pnl"`
	Notes      string  `db:"notes"`
}

//go:embed trade.insert.sql
var insertTradeSQL string

func InsertTrade(ctx context.Context, driver *database.Driver, t *domain.Trade) (*Trade, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	trade := &Trade{
		ID:         database.NewID(),
		StrategyID: t.StrategyID,
		Symbol:     t.Symbol,
		Currency:   t.Currency,
		Direction:  string(t.Direction),
		Quantity:   t.Quantity,
		Price:      t.Price,
		ExecutedAt: database.Time2DB(t.ExecutedAt),
		Status:     string(t.Status),
		PnL:        t.PnL,
		Notes:      t.Notes,
	}
	err = tx.ExecTx(txCtx, insertTradeSQL,
		trade.ID,
		trade.StrategyID,
		trade.Symbol,
		trade.Currency,
		trade.Direction,
		trade.Quantity,
		trade.Price,
		trade.ExecutedAt,
		trade.Status,
		trade.PnL,
		trade.Notes)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return trade, nil
}

//go:embed trade.select_by_id.sql
var selectTradeByIDSQL string

func SelectTradeByID(ctx context.Context, driver *database.Driver, id string) (*Trade, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	trade := &Trade{
		ID: id,
	}
	row, err := tx.QueryRowTx(txCtx, selectTradeByIDSQL, trade.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, trade,
		"strategy_id",
		"symbol",
		"currency",
		"direction",
		"quantity",
		"price",
		"executed_at",
		"status",
		"pnl",
		"notes")
	if database.NoRows(err) {
		trade = nil
	} else if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return trade, nil
}

//go:embed trade.select_by_symbol.sql
var selectTradeBySymbolSQL string

func SelectTradesBySymbol(ctx context.Context, driver *database.Driver, symbol string) ([]*Trade, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectTradeBySymbolSQL, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	trades := make([]*Trade, 0)
	for rows.Next() {
		trade := &Trade{
			Symbol: symbol,
		}
		err = database.Scan(rows, trade)
		if err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return trades, nil
}

//go:embed trade.select_by_strategy_id.sql
var selectTradeByStrategyIDSQL string

func SelectTradesByStrategyID(ctx context.Context, driver *database.Driver, strategyID string) ([]*Trade, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectTradeByStrategyIDSQL, strategyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	trades := make([]*Trade, 0)
	for rows.Next() {
		trade := &Trade{
			StrategyID: strategyID,
		}
		err = database.Scan(rows, trade)
		if err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return trades, nil
}
