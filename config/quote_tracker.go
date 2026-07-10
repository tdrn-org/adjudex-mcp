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

package config

import "time"

type QuoteTrackerConfig struct {
	Currency     string                    `toml:"currency"`
	Demo         DemoQuoteTrackerConfig    `toml:"demo"`
	AlphaVantage AlphaVantageTrackerConfig `toml:"alphavantage"`
	TwelveData   TwelveDataTrackerConfig   `toml:"twelvedata"`
}

type DemoQuoteTrackerConfig struct {
	Enabled   bool          `toml:"enabled"`
	Online    bool          `toml:"online"`
	Frequency time.Duration `toml:"frequency"`
}

type AlphaVantageTrackerConfig struct {
	Enabled   bool          `toml:"enabled"`
	Online    bool          `toml:"online"`
	Frequency time.Duration `toml:"frequency"`
	APIKey    string        `toml:"api_key"`
}

type TwelveDataTrackerConfig struct {
	Enabled   bool          `toml:"enabled"`
	Online    bool          `toml:"online"`
	Frequency time.Duration `toml:"frequency"`
	APIKey    string        `toml:"api_key"`
}
