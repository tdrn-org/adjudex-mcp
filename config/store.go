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
	"github.com/tdrn-org/go-config-toml"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-database/memory"
	"github.com/tdrn-org/go-database/sqlite"
)

type StoreConfig struct {
	DatabaseType DatabaseType `toml:"type"`
	MemoryConfig struct {     /* no parameters */
	} `toml:"memory"`
	SQLiteConfig struct {
		File string `toml:"file"`
	} `toml:"sqlite"`
}

type DatabaseType database.Type

var databaseTypeMarshalMap map[DatabaseType]string = map[DatabaseType]string{
	DatabaseType(memory.Type): string(memory.Type),
	DatabaseType(sqlite.Type): string(sqlite.Type),
}

var databaseTypeUnmarshalMap map[string]DatabaseType = map[string]DatabaseType{
	string(memory.Type): DatabaseType(memory.Type),
	string(sqlite.Type): DatabaseType(sqlite.Type),
}

func (t DatabaseType) MarshalText() ([]byte, error) {
	return config.MarshalEnum(t, databaseTypeMarshalMap)
}

func (t *DatabaseType) UnmarshalText(text []byte) error {
	databaseType, err := config.UnmarshalEnum(databaseTypeUnmarshalMap, text)
	if err != nil {
		return nil
	}
	*t = databaseType
	return nil
}
