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

package trace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type Protocol string

const (
	ProtocolHttp Protocol = "http"
	ProtocolGRPC Protocol = "gRPC"
)

type Options struct {
	ServiceName   string
	Protocol      Protocol
	EndpointURL   *url.URL
	BatchTimeout  time.Duration
	ExportTimeout time.Duration
	Headers       map[string]string
}

type Provider struct {
	exporter      *otlptrace.Exporter
	traceProvider *tracesdk.TracerProvider
}

func (p *Provider) Shutdown(ctx context.Context) error {
	return errors.Join(p.traceProvider.Shutdown(ctx), p.exporter.Shutdown(ctx))
}

func Start(ctx context.Context, options *Options) (*Provider, error) {
	var exporter *otlptrace.Exporter
	var err error
	switch options.Protocol {
	case ProtocolHttp:
		exporter, err = newHttpExporter(ctx, options.EndpointURL, options.Headers)
	case ProtocolGRPC:
		exporter, err = newGRPCExporter(ctx, options.EndpointURL, options.Headers)
	default:
		err = fmt.Errorf("unexpected tracing protocol: %q", options.Protocol)
	}
	if err != nil {
		return nil, err
	}
	serviceName := resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(options.ServiceName))
	traceProvider := tracesdk.NewTracerProvider(withBatcher(exporter, options.BatchTimeout, options.ExportTimeout), tracesdk.WithResource(serviceName))
	otel.SetTracerProvider(traceProvider)
	provider := &Provider{
		exporter:      exporter,
		traceProvider: traceProvider,
	}
	return provider, nil
}

func withBatcher(exporter *otlptrace.Exporter, batchTimeout time.Duration, exportTimeout time.Duration) tracesdk.TracerProviderOption {
	opts := []tracesdk.BatchSpanProcessorOption{
		tracesdk.WithBatchTimeout(batchTimeout),
		tracesdk.WithExportTimeout(exportTimeout),
	}
	return tracesdk.WithBatcher(exporter, opts...)
}

func newHttpExporter(ctx context.Context, endpoint *url.URL, headers map[string]string) (*otlptrace.Exporter, error) {
	opts := make([]otlptracehttp.Option, 0)
	opts = append(opts, otlptracehttp.WithEndpointURL(endpoint.String()))
	if endpoint.Scheme == "http" {
		slog.Warn("using insecure tracing endpoint URL", slog.Any("endpoint", endpoint))
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	opts = append(opts, otlptracehttp.WithHeaders(headers))
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create http Exporter (cause: %w)", err)
	}
	return exporter, nil
}

func newGRPCExporter(ctx context.Context, endpoint *url.URL, headers map[string]string) (*otlptrace.Exporter, error) {
	opts := make([]otlptracegrpc.Option, 0)
	opts = append(opts, otlptracegrpc.WithEndpointURL(endpoint.String()))
	if endpoint.Scheme == "http" {
		slog.Warn("using insecure tracing gRPC endpoint", slog.Any("endpoint", endpoint))
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	opts = append(opts, otlptracegrpc.WithHeaders(headers))
	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC Exporter (cause: %w)", err)
	}
	return exporter, nil
}
