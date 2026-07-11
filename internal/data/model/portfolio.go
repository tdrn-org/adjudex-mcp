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
	driver      *database.Driver
	ID          string `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
	CreatedAt   int64  `db:"created_at"`
	UpdatedAt   int64  `db:"updated_at"`
}

//go:embed portfolio.insert.sql
var insertPortfolioSQL string

func InsertPortfolio(ctx context.Context, driver *database.Driver, p *domain.Portfolio) (*Portfolio, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	portfolio := &Portfolio{
		driver:      driver,
		ID:          database.NewID(),
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   database.Time2DB(tx.Now()),
		UpdatedAt:   database.Time2DB(tx.Now()),
	}
	err = tx.ExecTx(txCtx, insertPortfolioSQL,
		portfolio.ID,
		portfolio.Name,
		portfolio.Description,
		portfolio.CreatedAt,
		portfolio.UpdatedAt)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return portfolio, nil
}

//go:embed portfolio.update_by_id.2.sql
var updatePortfolioByID2SQL string

func TouchPortfolioByID(ctx context.Context, driver *database.Driver, portfolioID string) error {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, updatePortfolioByID2SQL,
		database.Time2DB(tx.Now()),
		portfolioID)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

//go:embed portfolio.select_by_id.sql
var selectPortfolioByIDSQL string

func SelectPortfolioByID(ctx context.Context, driver *database.Driver, id string) (*Portfolio, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	portfolio := &Portfolio{
		driver: driver,
		ID:     id,
	}
	row, err := tx.QueryRowTx(txCtx, selectPortfolioByIDSQL, portfolio.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, portfolio,
		"name",
		"description",
		"created_at",
		"updated_at")
	if database.NoRows(err) {
		portfolio = nil
	} else if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return portfolio, nil
}

//go:embed portfolio.select.sql
var selectPortfolioSQL string

func SelectPortfolios(ctx context.Context, driver *database.Driver) ([]*Portfolio, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectPortfolioSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	portfolios := make([]*Portfolio, 0)
	for rows.Next() {
		portfolio := &Portfolio{
			driver: driver,
		}
		err = database.Scan(rows, portfolio)
		if err != nil {
			return nil, err
		}
		portfolios = append(portfolios, portfolio)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return portfolios, nil
}

//go:embed portfolio.delete_by_id.sql
var deletePortfolioByIDSQL string

func DeletePortfolioByID(ctx context.Context, driver *database.Driver, id string) error {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, deletePortfolioByIDSQL, id)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
