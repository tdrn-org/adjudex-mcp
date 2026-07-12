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

type Quote struct {
	Symbol          string  `db:"symbol"`
	Timestamp       int64   `db:"timestamp"`
	Currency        string  `db:"currency"`
	Open            float64 `db:"open"`
	High            float64 `db:"high"`
	Low             float64 `db:"low"`
	Close           float64 `db:"close"`
	Price           float64 `db:"price"`
	Volume          int64   `db:"volume"`
	Source          string  `db:"source"`
	SourceTimestamp int64   `db:"source_timestamp"`
}

//go:embed quote.insert.sql
var insertQuoteSQL string

func InsertQuote(ctx context.Context, driver *database.Driver, q *domain.Quote) (*Quote, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	quote := &Quote{
		Symbol:          q.Symbol,
		Timestamp:       database.Time2DB(q.Timestamp),
		Currency:        q.Currency,
		Open:            q.Open,
		High:            q.High,
		Low:             q.Low,
		Close:           q.Close,
		Price:           q.Price,
		Volume:          q.Volume,
		Source:          q.Source,
		SourceTimestamp: database.Time2DB(q.SourceTimestamp),
	}
	err = tx.ExecTx(txCtx, insertQuoteSQL,
		quote.Symbol,
		quote.Timestamp,
		quote.Currency,
		quote.Open,
		quote.High,
		quote.Low,
		quote.Close,
		quote.Price,
		quote.Volume,
		quote.Source,
		quote.SourceTimestamp)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return quote, nil
}

//go:embed quote.select.sql
var selectQuoteSQL string

func SelectLatestQuote(ctx context.Context, driver *database.Driver, symbol string) (*Quote, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	quote := &Quote{
		Symbol: symbol,
	}
	row, err := tx.QueryRowTx(txCtx, selectQuoteSQL, quote.Symbol)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, quote,
		"timestamp",
		"currency",
		"open",
		"high",
		"low",
		"close",
		"price",
		"volume",
		"source",
		"source_timestamp")
	if database.NoRows(err) {
		quote = nil
	} else if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return quote, nil
}

//go:embed quote.select_by_time_range.sql
var selectQuoteByTimerangeSQL string

func SelectQuotes(ctx context.Context, driver *database.Driver, symbol string, from, to time.Time) ([]*Quote, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectQuoteByTimerangeSQL, symbol, database.Time2DB(from), database.Time2DB(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	quotes := make([]*Quote, 0)
	for rows.Next() {
		quote := &Quote{
			Symbol: symbol,
		}
		err = database.Scan(rows, quote)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, quote)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return quotes, nil
}

//go:embed quote.delete_by_pk.sql
var deleteQuoteByPKSQL string

func DeleteQuoteByPK(ctx context.Context, driver *database.Driver, symbol string, timestamp time.Time) error {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, deleteQuoteByPKSQL, symbol, database.Time2DB(timestamp))
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
