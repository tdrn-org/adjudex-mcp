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

package tools

import (
	"context"
	"fmt"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

// --- Tool: alert_create ---

// AlertCreateArgs are the parameters for alert_create.
type AlertCreateArgs struct {
	Name      string                `json:"name"`
	Symbol    string                `json:"symbol"`
	Condition string                `json:"condition"` // price_above, price_below, rsi_above, rsi_below, sma_cross_above, sma_cross_below, volume_spike
	Threshold float64               `json:"threshold"`
	Indicator *domain.IndicatorSpec `json:"indicator,omitempty"` // nil for price-based alerts
}

// AlertCreate creates a new alert trigger.
// Store: domain.AlertStore.CreateAlert
func AlertCreate(ctx context.Context, store domain.AlertStore, args AlertCreateArgs) (*domain.Alert, error) {
	a := &domain.Alert{
		Name:      args.Name,
		Symbol:    args.Symbol,
		Condition: domain.AlertCondition(args.Condition),
		Threshold: args.Threshold,
		Indicator: args.Indicator,
	}
	if err := store.CreateAlert(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// --- Tool: alert_list ---

// AlertListArgs are the parameters for alert_list.
type AlertListArgs struct {
	Symbol string `json:"symbol,omitempty"` // optional filter by symbol
}

// AlertList returns all alerts, optionally filtered by symbol.
// Store: domain.AlertStore.ListAlerts
func AlertList(ctx context.Context, store domain.AlertStore, args AlertListArgs) ([]domain.Alert, error) {
	if args.Symbol != "" {
		return store.ListAlerts(ctx, args.Symbol)
	}
	return store.ListAlerts(ctx, "")
}

// --- Tool: alert_acknowledge ---

// AlertAcknowledgeArgs are the parameters for alert_acknowledge.
type AlertAcknowledgeArgs struct {
	ID string `json:"id"`
}

// AlertAcknowledge marks an alert as acknowledged.
// State machine: Armed/Triggered → Acknowledged.
// Store: domain.AlertStore.GetAlert → domain.Alert.Acknowledge() → domain.AlertStore.UpdateAlert
func AlertAcknowledge(ctx context.Context, store domain.AlertStore, args AlertAcknowledgeArgs) (*domain.Alert, error) {
	a, err := store.GetAlert(ctx, args.ID)
	if err != nil {
		return nil, fmt.Errorf("alert_acknowledge: %w", err)
	}
	a.Acknowledge()
	if err := store.UpdateAlert(ctx, a); err != nil {
		return nil, fmt.Errorf("alert_acknowledge: update: %w", err)
	}
	return a, nil
}

// --- Tool: alert_delete ---

// AlertDeleteArgs are the parameters for alert_delete.
type AlertDeleteArgs struct {
	ID string `json:"id"`
}

// AlertDelete removes an alert.
// Store: domain.AlertStore.DeleteAlert
func AlertDelete(ctx context.Context, store domain.AlertStore, args AlertDeleteArgs) error {
	return store.DeleteAlert(ctx, args.ID)
}
