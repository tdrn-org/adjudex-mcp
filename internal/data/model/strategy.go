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

func InsertStrategy(ctx context.Context, tx *database.Tx, strategy *domain.Strategy) (*Strategy, error) {
	s := &Strategy{
		ID:           database.NewID(),
		Name:         strategy.Name,
		Description:  strategy.Description,
		RSIPeriod:    strategy.Parameters.RSIPeriod,
		RSIThreshold: strategy.Parameters.RSIThreshold,
		SMAPeriod:    strategy.Parameters.SMAPeriod,
		SMATrigger:   strategy.Parameters.SMATrigger,
		MaxPosition:  strategy.Parameters.MaxPosition,
		StopLoss:     strategy.Parameters.StopLoss,
		CreatedAt:    database.Time2DB(tx.Now()),
		UpdatedAt:    database.Time2DB(tx.Now()),
	}
	err := tx.ExecTx(ctx, insertStrategySQL,
		s.ID,
		s.Name,
		s.Description,
		s.RSIPeriod,
		s.RSIThreshold,
		s.SMAPeriod,
		s.SMATrigger,
		s.MaxPosition,
		s.StopLoss,
		s.CreatedAt,
		s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

//go:embed strategy.select_by_id.sql
var selectStrategyByIDSQL string

func SelectStrategyByID(ctx context.Context, tx *database.Tx, id string) (*Strategy, error) {
	s := &Strategy{
		ID: id,
	}
	row, err := tx.QueryRowTx(ctx, selectStrategyByIDSQL, s.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, s,
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
		s = nil
	} else if err != nil {
		return nil, err
	}
	return s, nil
}

//go:embed strategy.select.sql
var selectStrategySQL string

func SelectStrategies(ctx context.Context, tx *database.Tx) ([]*Strategy, error) {
	rows, err := tx.QueryTx(ctx, selectStrategySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ss := make([]*Strategy, 0)
	for rows.Next() {
		s := &Strategy{}
		err = database.Scan(rows, s)
		if err != nil {
			return nil, err
		}
		ss = append(ss, s)
	}
	return ss, nil
}

//go:embed strategy.delete_by_id.sql
var deleteStrategyByIDSQL string

func DeleteStrategyByID(ctx context.Context, tx *database.Tx, id string) error {
	return tx.ExecTx(ctx, deleteStrategyByIDSQL, id)
}
