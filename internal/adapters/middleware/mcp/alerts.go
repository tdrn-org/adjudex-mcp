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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

func addAlertTools(server *mcp.Server, runtime Runtime) {
	addAlertCreateTool(server, runtime)
	addAlertGetTool(server, runtime)
	addAlertListTool(server, runtime)
	addAlertUpdateTool(server, runtime)
	addAlertDeleteTool(server, runtime)
}

func addAlertCreateTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "alert_create",
		Description: "Creates a new alert for a symbol. The alert fires when its condition is met on the next quote update.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"A human-readable label for the alert."},"symbol":{"type":"string","description":"The ticker symbol to watch."},"condition":{"type":"string","description":"Alert condition: price_above, price_below, rsi_above, rsi_below, sma_cross_above, sma_cross_below, volume_spike."},"threshold":{"type":"number","description":"The trigger value (e.g. 30.0 for RSI < 30)."},"indicator_type":{"type":"string","description":"Optional indicator type (sma, ema, rsi, macd). Required for indicator-based conditions."},"indicator_period":{"type":"integer","description":"Optional indicator period (e.g. 14 for RSI-14). Required for indicator-based conditions."}},"required":["name","symbol","condition","threshold"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name            string  `json:"name"`
			Symbol          string  `json:"symbol"`
			Condition       string  `json:"condition"`
			Threshold       float64 `json:"threshold"`
			IndicatorType   *string `json:"indicator_type"`
			IndicatorPeriod *int    `json:"indicator_period"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		condition := domain.AlertCondition(args.Condition)
		switch condition {
		case domain.AlertPriceAbove, domain.AlertPriceBelow, domain.AlertRSIAbove, domain.AlertRSIBelow,
			domain.AlertSMACrossAbove, domain.AlertSMACrossBelow, domain.AlertVolumeSpike:
		default:
			return nil, fmt.Errorf("unknown condition %q", args.Condition)
		}

		var indicator *domain.IndicatorSpec
		if args.IndicatorType != nil && args.IndicatorPeriod != nil {
			indicator = &domain.IndicatorSpec{
				Type:   domain.IndicatorType(*args.IndicatorType),
				Period: *args.IndicatorPeriod,
			}
		}

		a := &domain.Alert{
			Name:      args.Name,
			Symbol:    args.Symbol,
			Condition: condition,
			Threshold: args.Threshold,
			Indicator: indicator,
			State:     domain.AlertStateArmed,
		}
		if err := runtime.DataStore().CreateAlert(ctx, a); err != nil {
			return nil, fmt.Errorf("creating alert: %w", err)
		}
		return newToolResult(a)
	})
}

func addAlertGetTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "alert_get",
		Description: "Gets the full alert details for the given ID.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"ID of the alert to return."}},"required":["id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		a, err := runtime.DataStore().GetAlert(ctx, args.ID)
		if err != nil {
			return nil, fmt.Errorf("getting alert %q: %w", args.ID, err)
		}
		return newToolResult(a)
	})
}

func addAlertListTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "alert_list",
		Description: "Lists alerts. Optionally filter by symbol or state (armed, triggered, acknowledged, disabled).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"symbol":{"type":"string","description":"Optional symbol filter."},"state":{"type":"string","description":"Optional state filter (armed, triggered, acknowledged, disabled)."}}}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Symbol string `json:"symbol"`
			State  string `json:"state"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		if args.State == "armed" {
			alerts, err := runtime.DataStore().ListArmedAlerts(ctx)
			if err != nil {
				return nil, fmt.Errorf("listing armed alerts: %w", err)
			}
			return newToolResult(alerts)
		}

		alerts, err := runtime.DataStore().ListAlerts(ctx, args.Symbol)
		if err != nil {
			return nil, fmt.Errorf("listing alerts: %w", err)
		}

		if args.State != "" {
			filtered := make([]domain.Alert, 0)
			for _, a := range alerts {
				if string(a.State) == args.State {
					filtered = append(filtered, a)
				}
			}
			alerts = filtered
		}

		return newToolResult(alerts)
	})
}

func addAlertUpdateTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "alert_update",
		Description: "Updates an existing alert. Only the provided fields are updated (PATCH semantics). Use to acknowledge, rearm, or disable.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"ID of the alert to update."},"state":{"type":"string","description":"New state: armed, triggered, acknowledged, disabled."},"threshold":{"type":"number","description":"New threshold value."},"message":{"type":"string","description":"New message."}},"required":["id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID        string   `json:"id"`
			State     *string  `json:"state"`
			Threshold *float64 `json:"threshold"`
			Message   *string  `json:"message"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		a, err := runtime.DataStore().GetAlert(ctx, args.ID)
		if err != nil {
			return nil, fmt.Errorf("getting alert %q: %w", args.ID, err)
		}

		if args.State != nil {
			state := domain.AlertState(*args.State)
			switch state {
			case domain.AlertStateArmed:
				a.Rearm()
			case domain.AlertStateTriggered:
				now := time.Now()
				a.Fire(now, a.Message)
			case domain.AlertStateAcknowledged:
				a.Acknowledge()
			case domain.AlertStateDisabled:
				a.State = domain.AlertStateDisabled
			default:
				return nil, fmt.Errorf("unknown state %q", *args.State)
			}
		}
		if args.Threshold != nil {
			a.Threshold = *args.Threshold
		}
		if args.Message != nil {
			a.Message = *args.Message
		}

		if err := runtime.DataStore().UpdateAlert(ctx, a); err != nil {
			return nil, fmt.Errorf("updating alert %q: %w", args.ID, err)
		}
		return newToolResult(a)
	})
}

func addAlertDeleteTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "alert_delete",
		Description: "Deletes an alert.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"ID of the alert to delete."}},"required":["id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		if err := runtime.DataStore().DeleteAlert(ctx, args.ID); err != nil {
			return nil, fmt.Errorf("deleting alert %q: %w", args.ID, err)
		}
		return newToolResult(map[string]string{"status": "deleted", "id": args.ID})
	})
}
