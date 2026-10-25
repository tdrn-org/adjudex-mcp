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
	"github.com/tdrn-org/adjudex-mcp/internal/trace"
	"github.com/tdrn-org/go-config-toml"
)

type TracingConfig struct {
	Enabled       bool                `toml:"enabled"`
	Protocol      TracingProtocol     `toml:"protocol"`
	EndpointURL   config.URLSpec      `toml:"endpoint_url"`
	BatchTimeout  config.DurationSpec `toml:"batch_timeout"`
	ExportTimeout config.DurationSpec `toml:"export_timeout"`
	Headers       map[string]string   `toml:"headers"`
}

type TracingProtocol trace.Protocol

var TracingProtocolMarshalMap map[TracingProtocol]string = map[TracingProtocol]string{
	TracingProtocol(trace.ProtocolHttp): string(trace.ProtocolHttp),
	TracingProtocol(trace.ProtocolGRPC): string(trace.ProtocolGRPC),
}

var TracingProtocolUnmarshalMap map[string]TracingProtocol = map[string]TracingProtocol{
	string(trace.ProtocolHttp): TracingProtocol(trace.ProtocolHttp),
	string(trace.ProtocolGRPC): TracingProtocol(trace.ProtocolGRPC),
}

func (l TracingProtocol) String() string {
	return TracingProtocolMarshalMap[l]
}

func (l TracingProtocol) MarshalText() ([]byte, error) {
	return config.MarshalEnum(l, TracingProtocolMarshalMap)
}

func (l *TracingProtocol) UnmarshalText(text []byte) error {
	TracingProtocol, err := config.UnmarshalEnum(TracingProtocolUnmarshalMap, text)
	if err != nil {
		return err
	}
	*l = TracingProtocol
	return nil
}
