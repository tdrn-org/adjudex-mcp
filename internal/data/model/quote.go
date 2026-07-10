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

type Quote struct {
	//driver    *database.Driver
	Id        string  `db:"id"`
	Timestamp int64   `db:"id"`
	Open      float64 `db:"open"`
	High      float64 `db:"high"`
	Low       float64 `db:"low"`
	Close     float64 `db:"close"`
	Volume    int64   `db:"volume"`
	Source    string  `db:"source"`
}
