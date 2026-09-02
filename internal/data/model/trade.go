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

func InsertTrade(ctx context.Context, tx *database.Tx, trade *domain.Trade) (*Trade, error) {
	t := &Trade{
		ID:         database.NewID(),
		StrategyID: trade.StrategyID,
		Symbol:     trade.Symbol,
		Currency:   trade.Currency,
		Direction:  string(trade.Direction),
		Quantity:   trade.Quantity,
		Price:      trade.Price,
		ExecutedAt: database.Time2DB(trade.ExecutedAt),
		Status:     string(trade.Status),
		PnL:        trade.PnL,
		Notes:      trade.Notes,
	}
	err := tx.ExecTx(ctx, insertTradeSQL,
		t.ID,
		t.StrategyID,
		t.Symbol,
		t.Currency,
		t.Direction,
		t.Quantity,
		t.Price,
		t.ExecutedAt,
		t.Status,
		t.PnL,
		t.Notes)
	if err != nil {
		return nil, err
	}
	return t, nil
}

//go:embed trade.select_by_id.sql
var selectTradeByIDSQL string

func SelectTradeByID(ctx context.Context, tx *database.Tx, id string) (*Trade, error) {
	t := &Trade{
		ID: id,
	}
	row, err := tx.QueryRowTx(ctx, selectTradeByIDSQL, t.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, t,
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
		t = nil
	} else if err != nil {
		return nil, err
	}
	return t, nil
}

//go:embed trade.select_by_symbol.sql
var selectTradeBySymbolSQL string

func SelectTradesBySymbol(ctx context.Context, tx *database.Tx, symbol string) ([]*Trade, error) {
	rows, err := tx.QueryTx(ctx, selectTradeBySymbolSQL, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ts := make([]*Trade, 0)
	for rows.Next() {
		t := &Trade{
			Symbol: symbol,
		}
		err = database.Scan(rows, t)
		if err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, nil
}

//go:embed trade.select_by_strategy_id.sql
var selectTradeByStrategyIDSQL string

func SelectTradesByStrategyID(ctx context.Context, tx *database.Tx, strategyID string) ([]*Trade, error) {
	rows, err := tx.QueryTx(ctx, selectTradeByStrategyIDSQL, strategyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ts := make([]*Trade, 0)
	for rows.Next() {
		t := &Trade{
			StrategyID: strategyID,
		}
		err = database.Scan(rows, t)
		if err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, nil
}
