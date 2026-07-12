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

type Strategy struct {
	ID           string  `db:"id"`
	Name         string  `db:"name"`
	Description  string  `db:"description"`
	RSIPeriod    int     `db:"rsi_period"`
	RSIThreshold float64 `db:"rsi_threshold"`
	SMAPeriod    int     `db:"sma_period"`
	SMATrigger   float64 `db:"sma_trigger"`
	MaxPosition  float64 `db:"max_position"`
	StopLoss     float64 `db:"stop_loss"`
	CreatedAt    int64   `db:"created_at"`
	UpdatedAt    int64   `db:"updated_at"`
}

//go:embed strategy.insert.sql
var insertStrategySQL string

func InsertStrategy(ctx context.Context, driver *database.Driver, st *domain.Strategy) (*Strategy, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	strategy := &Strategy{
		ID:           database.NewID(),
		Name:         st.Name,
		Description:  st.Description,
		RSIPeriod:    st.Parameters.RSIPeriod,
		RSIThreshold: st.Parameters.RSIThreshold,
		SMAPeriod:    st.Parameters.SMAPeriod,
		SMATrigger:   st.Parameters.SMATrigger,
		MaxPosition:  st.Parameters.MaxPosition,
		StopLoss:     st.Parameters.StopLoss,
		CreatedAt:    database.Time2DB(tx.Now()),
		UpdatedAt:    database.Time2DB(tx.Now()),
	}
	err = tx.ExecTx(txCtx, insertStrategySQL,
		strategy.ID,
		strategy.Name,
		strategy.Description,
		strategy.RSIPeriod,
		strategy.RSIThreshold,
		strategy.SMAPeriod,
		strategy.SMATrigger,
		strategy.MaxPosition,
		strategy.StopLoss,
		strategy.CreatedAt,
		strategy.UpdatedAt)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return strategy, nil
}

//go:embed strategy.select_by_id.sql
var selectStrategyByIDSQL string

func SelectStrategyByID(ctx context.Context, driver *database.Driver, id string) (*Strategy, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	strategy := &Strategy{
		ID: id,
	}
	row, err := tx.QueryRowTx(txCtx, selectStrategyByIDSQL, strategy.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, strategy,
		"name",
		"description",
		"rsi_period",
		"rsi_threshold",
		"sma_period",
		"sma_trigger",
		"max_position",
		"stop_loss",
		"created_at",
		"updated_at")
	if database.NoRows(err) {
		strategy = nil
	} else if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return strategy, nil
}

//go:embed strategy.select.sql
var selectStrategySQL string

func SelectStrategies(ctx context.Context, driver *database.Driver) ([]*Strategy, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectStrategySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	strategies := make([]*Strategy, 0)
	for rows.Next() {
		strategy := &Strategy{}
		err = database.Scan(rows, strategy)
		if err != nil {
			return nil, err
		}
		strategies = append(strategies, strategy)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return strategies, nil
}

//go:embed strategy.delete_by_id.sql
var deleteStrategyByIDSQL string

func DeleteStrategyByID(ctx context.Context, driver *database.Driver, id string) error {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, deleteStrategyByIDSQL, id)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
