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

package domain

import "time"

// AlertCondition defines the trigger type for an alert.
type AlertCondition string

const (
	AlertPriceAbove    AlertCondition = "price_above"
	AlertPriceBelow    AlertCondition = "price_below"
	AlertRSIAbove      AlertCondition = "rsi_above"
	AlertRSIBelow      AlertCondition = "rsi_below"
	AlertSMACrossAbove AlertCondition = "sma_cross_above"
	AlertSMACrossBelow AlertCondition = "sma_cross_below"
	AlertVolumeSpike   AlertCondition = "volume_spike"
)

// AlertState tracks the lifecycle of an alert.
type AlertState string

const (
	AlertStateArmed        AlertState = "armed"
	AlertStateTriggered    AlertState = "triggered"
	AlertStateAcknowledged AlertState = "acknowledged"
	AlertStateDisabled     AlertState = "disabled"
)

// IndicatorSpec describes which indicator an alert watches.
// nil means the alert is price-based (not indicator-based).
type IndicatorSpec struct {
	Type   IndicatorType `json:"type"`
	Period int           `json:"period"`
}

// Alert represents a notification trigger attached to a security.
// It is evaluated on every quote update and fires when its condition is met.
type Alert struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	Condition   AlertCondition `json:"condition"`
	Threshold   float64        `json:"threshold"`
	Indicator   *IndicatorSpec `json:"indicator,omitempty"`
	State       AlertState     `json:"state"`
	TriggeredAt *time.Time     `json:"triggered_at,omitempty"`
	Message     string         `json:"message,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Evaluate checks whether the alert should fire given a current value.
// For price-based alerts, value is the current price.
// For indicator-based alerts, value is the indicator reading.
func (a *Alert) Evaluate(value float64) bool {
	if a.State != AlertStateArmed {
		return false
	}
	switch a.Condition {
	case AlertPriceAbove, AlertRSIAbove, AlertSMACrossAbove, AlertVolumeSpike:
		return value > a.Threshold
	case AlertPriceBelow, AlertRSIBelow, AlertSMACrossBelow:
		return value < a.Threshold
	default:
		return false
	}
}

// Fire transitions the alert to triggered state.
func (a *Alert) Fire(now time.Time, message string) {
	a.State = AlertStateTriggered
	a.TriggeredAt = &now
	a.Message = message
}

// Acknowledge marks the alert as seen.
func (a *Alert) Acknowledge() {
	a.State = AlertStateAcknowledged
}

// Rearm resets the alert to armed for reuse.
func (a *Alert) Rearm() {
	a.State = AlertStateArmed
	a.TriggeredAt = nil
	a.Message = ""
}
