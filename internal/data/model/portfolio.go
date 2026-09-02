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

type Portfolio struct {
	ID          string `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
	CreatedAt   int64  `db:"created_at"`
	UpdatedAt   int64  `db:"updated_at"`
}

//go:embed portfolio.insert.sql
var insertPortfolioSQL string

func InsertPortfolio(ctx context.Context, tx *database.Tx, portfolio *domain.Portfolio) (*Portfolio, error) {
	p := &Portfolio{
		ID:          database.NewID(),
		Name:        portfolio.Name,
		Description: portfolio.Description,
		CreatedAt:   database.Time2DB(tx.Now()),
		UpdatedAt:   database.Time2DB(tx.Now()),
	}
	err := tx.ExecTx(ctx, insertPortfolioSQL,
		p.ID,
		p.Name,
		p.Description,
		p.CreatedAt,
		p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

//go:embed portfolio.update_by_id.2.sql
var updatePortfolioByID2SQL string

func TouchPortfolioByID(ctx context.Context, tx *database.Tx, id string) error {
	return tx.ExecTx(ctx, updatePortfolioByID2SQL,
		database.Time2DB(tx.Now()),
		id)
}

//go:embed portfolio.select_by_id.sql
var selectPortfolioByIDSQL string

func SelectPortfolioByID(ctx context.Context, tx *database.Tx, id string) (*Portfolio, error) {
	p := &Portfolio{
		ID: id,
	}
	row, err := tx.QueryRowTx(ctx, selectPortfolioByIDSQL, p.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, p,
		"name",
		"description",
		"created_at",
		"updated_at")
	if database.NoRows(err) {
		p = nil
	} else if err != nil {
		return nil, err
	}
	return p, nil
}

//go:embed portfolio.select.sql
var selectPortfolioSQL string

func SelectPortfolios(ctx context.Context, tx *database.Tx) ([]*Portfolio, error) {
	rows, err := tx.QueryTx(ctx, selectPortfolioSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ps := make([]*Portfolio, 0)
	for rows.Next() {
		p := &Portfolio{}
		err = database.Scan(rows, p)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

//go:embed portfolio.delete_by_id.sql
var deletePortfolioByIDSQL string

func DeletePortfolioByID(ctx context.Context, tx *database.Tx, id string) error {
	return tx.ExecTx(ctx, deletePortfolioByIDSQL, id)
}
