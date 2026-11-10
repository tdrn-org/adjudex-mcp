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

package notification

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

type alertParams struct {
	Prefix        string
	Name          string
	Instrument    string
	Listing       string
	Condition     string
	ConditionText string
	Threshold     string
	Value         string
	Message       string
}

func newAlertParams(prefix string, alert *domain.Alert, quote *domain.Quote) *alertParams {
	return &alertParams{
		Prefix:        prefix,
		Name:          alert.Name,
		Instrument:    quote.Listing.InstrumentName(),
		Listing:       quote.Listing.ID(),
		Condition:     string(alert.Condition),
		ConditionText: conditionTextMap[alert.Condition],
		Threshold:     fmt.Sprintf("%.2f %s", alert.Threshold, alert.Currency),
		Value:         fmt.Sprintf("%.2f %s", quote.Price, quote.Currency),
		Message:       alert.Message,
	}
}

//go:embed condition_text.json
var conditionTextMapData []byte
var conditionTextMap map[domain.AlertCondition]string

func init() {
	err := json.Unmarshal(conditionTextMapData, &conditionTextMap)
	if err != nil {
		panic(err)
	}
}
