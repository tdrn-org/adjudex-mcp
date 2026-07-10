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

type Strategy struct {
	Id           string  `db:"id"`
	Name         string  `db:"name"`
	Description  string  `db:"description"`
	RsiPeriod    int     `db:"rsi_period"`
	RsiThreshold float64 `db:"rsi_threshold"`
	SmaPeriod    int     `db:"sma_period"`
	SmaTrigger   float64 `db:"sma_trigger"`
	MaxPosition  float64 `db:"max_position"`
	StopLoss     float64 `db:"stop_loss"`
	CreatedAt    int64   `db:"created_at"`
	UpdatedAt    int64   `db:"updated_at"`
}
