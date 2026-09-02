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

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-finance"
)

type Symbol struct {
	ID       string `db:"id"`
	Exchange string `db:"exchange"`
	Ticker   string `db:"ticker"`
	ISIN     string `db:"isin"`
	WKN      string `db:"wkn"`
	FIGI     string `db:"figi"`
	Name     string `db:"name"`
	Type     string `db:"type"`
}

func (s *Symbol) MergeSymbol(sym *finance.Symbol) {
	if s.Exchange == "" {
		s.Exchange = sym.Exchange
	}
	if s.Ticker == "" {
		s.Ticker = sym.Ticker
	}
	if s.ISIN == "" {
		s.ISIN = sym.ISIN
	}
	if s.WKN == "" {
		s.WKN = sym.WKN
	}
	if s.FIGI == "" {
		s.FIGI = sym.FIGI
	}
	if s.Name == "" {
		s.Name = sym.Name
	}
	if s.Type == "" {
		s.Type = string(sym.Type)
	}
}

//go:embed symbol.insert.sql
var insertSymbolSQL string

func InsertSymbol(ctx context.Context, tx *database.Tx, s *finance.Symbol) (*Symbol, error) {
	symbol := &Symbol{
		ID:       database.NewID(),
		Exchange: s.Exchange,
		Ticker:   s.Ticker,
		ISIN:     s.ISIN,
		WKN:      s.WKN,
		FIGI:     s.FIGI,
		Name:     s.Name,
		Type:     string(s.Type),
	}
	err := tx.ExecTx(ctx, insertSymbolSQL,
		symbol.ID,
		symbol.Exchange,
		symbol.Ticker,
		symbol.ISIN,
		symbol.WKN,
		symbol.FIGI,
		symbol.Name,
		symbol.Type)
	if err != nil {
		return nil, err
	}
	return symbol, nil
}

//go:embed symbol.select_by_id.sql
var selectSymbolByIDSQL string

func SelectSymbolByID(ctx context.Context, tx *database.Tx, id string) (*Symbol, error) {
	symbol := &Symbol{
		ID: id,
	}
	row, err := tx.QueryRowTx(ctx, selectSymbolByIDSQL, symbol.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, symbol,
		"exchange",
		"ticker",
		"isin",
		"wkn",
		"figi",
		"name",
		"symbol")
	if database.NoRows(err) {
		symbol = nil
	} else if err != nil {
		return nil, err
	}
	return symbol, nil
}

//go:embed symbol.select_by_symbol.sql
var selectSymbolBySymbolSQL string

func SelectSymbolBySymbol(ctx context.Context, tx *database.Tx, s *finance.Symbol) (*Symbol, error) {
	symbol := &Symbol{}
	row, err := tx.QueryRowTx(ctx, selectSymbolBySymbolSQL,
		s.Exchange,
		s.Ticker,
		s.ISIN,
		s.WKN,
		s.FIGI)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, symbol,
		"id",
		"exchange",
		"ticker",
		"isin",
		"wkn",
		"figi",
		"name",
		"type")
	if database.NoRows(err) {
		symbol = nil
	} else if err != nil {
		return nil, err
	}
	return symbol, nil
}

//go:embed symbol.update_by_id.sql
var updateSymbolByIDSQL string

func (s *Symbol) Update(ctx context.Context, tx *database.Tx) error {
	err := tx.ExecTx(ctx, updateSymbolByIDSQL,
		s.Exchange,
		s.Ticker,
		s.ISIN,
		s.WKN,
		s.FIGI,
		s.Name,
		s.Type,
		s.ID)
	if err != nil {
		return err
	}
	return nil
}
