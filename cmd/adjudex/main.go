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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/api"
	"github.com/tdrn-org/adjudex-mcp/internal/mcp"
	"github.com/tdrn-org/adjudex-mcp/internal/quote"
	"github.com/tdrn-org/adjudex-mcp/internal/stock"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/alphavantage"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/multi"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/twelvedata"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/yahoo"
	"github.com/tdrn-org/adjudex-mcp/internal/store"
	"github.com/tdrn-org/adjudex-mcp/internal/web"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/go-httpserver/certificate"
)

// defaults
const (
	defaultTransport = "stdio"
	defaultAddr      = ":8080"
	defaultDB        = "adjudex.db"
)

// env overrides
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	transport := flag.String("transport", env("ADJUDEX_TRANSPORT", defaultTransport),
		"Transport mode: stdio, sse, or http")
	addr := flag.String("addr", env("ADJUDEX_ADDR", defaultAddr),
		"Listen address (SSE and HTTP modes)")
	dbPath := flag.String("db", env("ADJUDEX_DB", defaultDB),
		"Path to SQLite database file (use \"memory\" for in-memory DB)")
	tlsCert := flag.String("tls-cert", env("ADJUDEX_TLS_CERT", ""),
		"Path to TLS certificate file (enables HTTPS)")
	tlsKey := flag.String("tls-key", env("ADJUDEX_TLS_KEY", ""),
		"Path to TLS key file (enables HTTPS)")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "adjudex v0.3.0 — Agent juDy eXchange\n")

	// Database
	var dbConfig database.Config
	if *dbPath == "memory" {
		fmt.Fprintf(os.Stderr, "Using in-memory database (data will be lost on exit)\n")
		dbConfig = store.MemoryConfig()
	} else {
		dbConfig = store.Config(*dbPath)
	}
	dbDriver, err := database.Open(dbConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: open database: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if _, _, err := dbDriver.UpdateSchema(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: migrate database: %v\n", err)
		os.Exit(1)
	}

	s := store.NewStore(dbDriver)
	storeSet := mcp.StoreSet{
		Portfolio: s,
		Quote:     s,
		Alert:     s,
		Trade:     s,
		Strategy:  s,
	}

	server := mcp.New(storeSet)

	switch *transport {
	case "stdio":
		fmt.Fprintf(os.Stderr, "MCP server starting on stdio...\n")
		if err := server.ServeStdio(); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: stdio serve: %v\n", err)
			os.Exit(1)
		}

	case "sse":
		fmt.Fprintf(os.Stderr, "MCP server starting on SSE %s...\n", *addr)
		if err := server.ServeSSE(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: sse serve: %v\n", err)
			os.Exit(1)
		}

	case "http":
		// Build TLS options if cert/key provided
		var opts []httpserver.OptionSetter
		scheme := "http"
		if *tlsCert != "" && *tlsKey != "" {
			opts = append(opts, httpserver.WithCertificateProvider(
				&certificate.FileCertificateProvider{
					CertFile: *tlsCert,
					KeyFile:  *tlsKey,
				},
			))
			scheme = "https"
		}

		fmt.Fprintf(os.Stderr, "REST API + SvelteKit frontend starting on %s://%s...\n", scheme, *addr)
		httpSrv, err := httpserver.Listen(ctx, "tcp", *addr, opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: listen: %v\n", err)
			os.Exit(1)
		}
		httpSrv.Handle("/", web.Handler())

		// Build provider chain: Twelve Data → Alpha Vantage → Yahoo mock
		var providers []stock.Provider
		if key := os.Getenv("ADJUDEX_TWELVEDATA_KEY"); key != "" {
			providers = append(providers, twelvedata.NewProvider(key))
			fmt.Fprintf(os.Stderr, "Provider 1: Twelve Data (live US stocks)\n")
		}
		if key := os.Getenv("ADJUDEX_ALPHAVANTAGE_KEY"); key != "" {
			providers = append(providers, alphavantage.NewProvider(key))
			fmt.Fprintf(os.Stderr, "Provider %d: Alpha Vantage (international)\n", len(providers))
		}
		providers = append(providers, yahoo.NewProvider())
		fmt.Fprintf(os.Stderr, "Provider %d: Yahoo mock (fallback)\n", len(providers))
		provider := multi.New(providers...)
		api.Router(httpSrv, storeSet, provider)

		// Background quote poller: fetches quotes for all portfolio symbols every 5 min,
		// with at least 1 min between polls of the same symbol (rate limiting).
		poller := quote.NewPoller(provider, s, s, 5*time.Minute, 1*time.Minute)
		ctxPoller, cancelPoller := context.WithCancel(ctx)
		defer cancelPoller()
		go poller.Run(ctxPoller)
		fmt.Fprintf(os.Stderr, "Background poller started (interval=5m, minInterval=1m)\n")

		// Graceful shutdown on SIGINT/SIGTERM
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Fprintf(os.Stderr, "\nShutting down...\n")
			cancelPoller()
		}()

		if err := httpSrv.Serve(); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: serve: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "FATAL: unknown transport %q (use stdio, sse, or http)\n", *transport)
		os.Exit(1)
	}
}
