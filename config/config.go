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

import (
	_ "embed"

	"github.com/tdrn-org/go-config-toml"
	"github.com/tdrn-org/go-finance"
	financeconfig "github.com/tdrn-org/go-finance/config"
)

type Config struct {
	Logging  LoggingConfig        `toml:"logging"`
	Server   ServerConfig         `toml:"server"`
	Store    StoreConfig          `toml:"store"`
	Provider financeconfig.Config `toml:"provider"`
	Display  DisplayConfig        `toml:"display"`
	Metrics  MetricsConfig        `toml:"metrics"`
	Tracing  TracingConfig        `toml:"tracing"`
}

// DisplayConfig holds the presentation preferences of the user interface.
type DisplayConfig struct {
	// Currency is the currency the user wants values presented in. It is a
	// preference of the CLIENT, not a default of the domain: requests without an
	// explicit currency fall back to it, but no calculation invents a currency
	// on its own. Denominations stay whatever their source reported.
	Currency finance.Currency `toml:"currency"`
}

//go:embed defaults.toml
var defaultsData []byte

func Default() (*Config, error) {
	return config.Defaults(&Config{}, defaultsData)
}

func Load(path string) (*Config, error) {
	return config.Load(&Config{}, path, defaultsData, false)
}
