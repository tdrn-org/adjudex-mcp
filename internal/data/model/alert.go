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
	driver          *database.Driver
	ID              string  `db:"id"`
	Name            string  `db:"name"`
	Symbol          string  `db:"symbol"`
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

func InsertAlert(ctx context.Context, driver *database.Driver, a *domain.Alert) (*Alert, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	var indicatorType *string
	var indicatorPeriod *int
	if a.Indicator != nil {
		indicatorType = (*string)(&a.Indicator.Type)
		indicatorPeriod = &a.Indicator.Period
	}
	alert := &Alert{
		driver:          driver,
		ID:              database.NewID(),
		Name:            a.Name,
		Symbol:          a.Symbol,
		Condition:       string(a.Condition),
		Threshold:       a.Threshold,
		IndicatorType:   indicatorType,
		IndicatorPeriod: indicatorPeriod,
		State:           string(a.State),
		TriggeredAt:     ptrTime(a.TriggeredAt),
		Message:         a.Message,
		CreatedAt:       database.Time2DB(tx.Now()),
		UpdatedAt:       database.Time2DB(tx.Now()),
	}
	err = tx.ExecTx(txCtx, insertAlertSQL,
		alert.ID,
		alert.Name,
		alert.Symbol,
		alert.Condition,
		alert.Threshold,
		alert.IndicatorType,
		alert.IndicatorPeriod,
		alert.State,
		alert.TriggeredAt,
		alert.Message,
		alert.CreatedAt,
		alert.UpdatedAt)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

//go:embed alert.update_by_id.sql
var updateAlertByIDSQL string

func UpdateAlert(ctx context.Context, driver *database.Driver, a *domain.Alert) (*Alert, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	var indicatorType *string
	var indicatorPeriod *int
	if a.Indicator != nil {
		indicatorType = (*string)(&a.Indicator.Type)
		indicatorPeriod = &a.Indicator.Period
	}
	alert := &Alert{
		ID:              database.NewID(),
		Name:            a.Name,
		Symbol:          a.Symbol,
		Condition:       string(a.Condition),
		Threshold:       a.Threshold,
		IndicatorType:   indicatorType,
		IndicatorPeriod: indicatorPeriod,
		State:           string(a.State),
		TriggeredAt:     ptrTime(a.TriggeredAt),
		Message:         a.Message,
		UpdatedAt:       database.Time2DB(tx.Now()),
	}
	err = tx.ExecTx(txCtx, updateAlertByIDSQL,
		alert.Name,
		alert.Symbol,
		alert.Condition,
		alert.Threshold,
		alert.IndicatorType,
		alert.IndicatorPeriod,
		alert.State,
		alert.TriggeredAt,
		alert.Message,
		alert.UpdatedAt,
		alert.ID)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

//go:embed alert.select_by_id.sql
var selectAlertByIDSQL string

func SelectAlertByID(ctx context.Context, driver *database.Driver, id string) (*Alert, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	alert := &Alert{
		driver: driver,
		ID:     id,
	}
	row, err := tx.QueryRowTx(txCtx, selectAlertByIDSQL, alert.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, alert,
		"name",
		"symbol",
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
		alert = nil
	} else if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

//go:embed alert.select_by_symbol.sql
var selectAlertBySymbolSQL string

func SelectAlertsBySymbol(ctx context.Context, driver *database.Driver, symbol string) ([]*Alert, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectAlertBySymbolSQL, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	alerts := make([]*Alert, 0)
	for rows.Next() {
		alert := &Alert{
			driver: driver,
			Symbol: symbol,
		}
		err = database.Scan(rows, alert)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

//go:embed alert.select_by_state.sql
var selectAlertByStateSQL string

func SelectAlertsByState(ctx context.Context, driver *database.Driver, state domain.AlertState) ([]*Alert, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	rows, err := tx.QueryTx(txCtx, selectAlertByStateSQL, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	alerts := make([]*Alert, 0)
	for rows.Next() {
		alert := &Alert{
			driver: driver,
			State:  string(state),
		}
		err = database.Scan(rows, alert)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

//go:embed alert.delete_by_id.sql
var deleteAlertByIDSQL string

func DeleteAlertByID(ctx context.Context, driver *database.Driver, id string) error {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, deleteAlertByIDSQL, id)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
