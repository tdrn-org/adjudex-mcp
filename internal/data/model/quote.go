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

func InsertQuote(ctx context.Context, tx *database.Tx, quote *domain.Quote) (*Quote, error) {
	q := &Quote{
		Symbol:          quote.Symbol,
		Timestamp:       database.Time2DB(quote.Timestamp),
		Currency:        quote.Currency,
		Open:            quote.Open,
		High:            quote.High,
		Low:             quote.Low,
		Close:           quote.Close,
		Price:           quote.Price,
		Volume:          quote.Volume,
		Source:          quote.Source,
		SourceTimestamp: database.Time2DB(quote.SourceTimestamp),
	}
	err := tx.ExecTx(ctx, insertQuoteSQL,
		q.Symbol,
		q.Timestamp,
		q.Currency,
		q.Open,
		q.High,
		q.Low,
		q.Close,
		q.Price,
		q.Volume,
		q.Source,
		q.SourceTimestamp)
	if err != nil {
		return nil, err
	}
	return q, nil
}

//go:embed quote.select.sql
var selectQuoteSQL string

func SelectLatestQuote(ctx context.Context, tx *database.Tx, symbol string) (*Quote, error) {
	q := &Quote{
		Symbol: symbol,
	}
	row, err := tx.QueryRowTx(ctx, selectQuoteSQL, q.Symbol)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, q,
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
		q = nil
	} else if err != nil {
		return nil, err
	}
	return q, nil
}

//go:embed quote.select_by_time_range.sql
var selectQuoteByTimerangeSQL string

func SelectQuotes(ctx context.Context, tx *database.Tx, symbol string, from, to time.Time) ([]*Quote, error) {
	rows, err := tx.QueryTx(ctx, selectQuoteByTimerangeSQL, symbol, database.Time2DB(from), database.Time2DB(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	qs := make([]*Quote, 0)
	for rows.Next() {
		q := &Quote{
			Symbol: symbol,
		}
		err = database.Scan(rows, q)
		if err != nil {
			return nil, err
		}
		qs = append(qs, q)
	}
	return qs, nil
}

//go:embed quote.delete_by_pk.sql
var deleteQuoteByPKSQL string

func DeleteQuoteByPK(ctx context.Context, tx *database.Tx, symbol string, timestamp time.Time) error {
	return tx.ExecTx(ctx, deleteQuoteByPKSQL, symbol, database.Time2DB(timestamp))
}
