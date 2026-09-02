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

type Alert struct {
	ID              string  `db:"id"`
	Name            string  `db:"name"`
	Symbol          string  `db:"symbol"`
	Currency        string  `db:"currency"`
	Condition       string  `db:"condition"`
	Threshold       float64 `db:"threshold"`
	IndicatorType   *string `db:"indicator_type"`
	IndicatorPeriod *int    `db:"indicator_period"`
	State           string  `db:"state"`
	TriggeredAt     *int64  `db:"triggered_at"`
	Message         string  `db:"message"`
	CreatedAt       int64   `db:"created_at"`
	UpdatedAt       int64   `db:"updated_at"`
}

//go:embed alert.insert.sql
var insertAlertSQL string

func InsertAlert(ctx context.Context, tx *database.Tx, alert *domain.Alert) (*Alert, error) {
	var indicatorType *string
	var indicatorPeriod *int
	if alert.Indicator != nil {
		indicatorType = stringPtr(string(alert.Indicator.Type))
		indicatorPeriod = &alert.Indicator.Period
	}
	a := &Alert{
		ID:              database.NewID(),
		Name:            alert.Name,
		Symbol:          alert.Symbol,
		Currency:        alert.Currency,
		Condition:       string(alert.Condition),
		Threshold:       alert.Threshold,
		IndicatorType:   indicatorType,
		IndicatorPeriod: indicatorPeriod,
		State:           string(alert.State),
		TriggeredAt:     ptrTime(alert.TriggeredAt),
		Message:         alert.Message,
		CreatedAt:       database.Time2DB(tx.Now()),
		UpdatedAt:       database.Time2DB(tx.Now()),
	}
	err := tx.ExecTx(ctx, insertAlertSQL,
		a.ID,
		a.Name,
		a.Symbol,
		a.Currency,
		a.Condition,
		a.Threshold,
		a.IndicatorType,
		a.IndicatorPeriod,
		a.State,
		a.TriggeredAt,
		a.Message,
		a.CreatedAt,
		a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

//go:embed alert.update_by_id.sql
var updateAlertByIDSQL string

func UpdateAlert(ctx context.Context, tx *database.Tx, alert *domain.Alert) (*Alert, error) {
	var indicatorType *string
	var indicatorPeriod *int
	if alert.Indicator != nil {
		indicatorType = stringPtr(string(alert.Indicator.Type))
		indicatorPeriod = &alert.Indicator.Period
	}
	a := &Alert{
		ID:              alert.ID,
		Name:            alert.Name,
		Symbol:          alert.Symbol,
		Currency:        alert.Currency,
		Condition:       string(alert.Condition),
		Threshold:       alert.Threshold,
		IndicatorType:   indicatorType,
		IndicatorPeriod: indicatorPeriod,
		State:           string(alert.State),
		TriggeredAt:     ptrTime(alert.TriggeredAt),
		Message:         alert.Message,
		UpdatedAt:       database.Time2DB(tx.Now()),
	}
	err := tx.ExecTx(ctx, updateAlertByIDSQL,
		a.Name,
		a.Symbol,
		a.Currency,
		a.Condition,
		a.Threshold,
		a.IndicatorType,
		a.IndicatorPeriod,
		a.State,
		a.TriggeredAt,
		a.Message,
		a.UpdatedAt,
		a.ID)
	if err != nil {
		return nil, err
	}
	return a, nil
}

//go:embed alert.select_by_id.sql
var selectAlertByIDSQL string

func SelectAlertByID(ctx context.Context, tx *database.Tx, id string) (*Alert, error) {
	a := &Alert{
		ID: id,
	}
	row, err := tx.QueryRowTx(ctx, selectAlertByIDSQL, a.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, a,
		"name",
		"symbol",
		"currency",
		"condition",
		"threshold",
		"indicator_type",
		"indicator_period",
		"state",
		"triggered_at",
		"message",
		"created_at",
		"updated_at")
	if database.NoRows(err) {
		a = nil
	} else if err != nil {
		return nil, err
	}
	return a, nil
}

//go:embed alert.select_by_symbol.sql
var selectAlertBySymbolSQL string

func SelectAlertsBySymbol(ctx context.Context, tx *database.Tx, symbol string) ([]*Alert, error) {
	rows, err := tx.QueryTx(ctx, selectAlertBySymbolSQL, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	as := make([]*Alert, 0)
	for rows.Next() {
		a := &Alert{
			Symbol: symbol,
		}
		err = database.Scan(rows, a)
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
	return as, nil
}

//go:embed alert.select_by_state.sql
var selectAlertByStateSQL string

func SelectAlertsByState(ctx context.Context, tx *database.Tx, state domain.AlertState) ([]*Alert, error) {
	rows, err := tx.QueryTx(ctx, selectAlertByStateSQL, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	as := make([]*Alert, 0)
	for rows.Next() {
		a := &Alert{
			State: string(state),
		}
		err = database.Scan(rows, a)
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
	return as, nil
}

//go:embed alert.delete_by_id.sql
var deleteAlertByIDSQL string

func DeleteAlertByID(ctx context.Context, tx *database.Tx, id string) error {
	return tx.ExecTx(ctx, deleteAlertByIDSQL, id)
}
