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

type Alert struct {
	Id              string  `db:"id"`
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
