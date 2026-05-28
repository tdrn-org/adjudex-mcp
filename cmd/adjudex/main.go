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

// Package main is the entry point for adjudex — Agent juDy eXchange.
// adjudex is an MCP server for tracking and analyzing stock prices,
// with an embedded SvelteKit web frontend.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/tdrn-org/adjudex-mcp/internal/api"
	"github.com/tdrn-org/adjudex-mcp/internal/mcp"
	"github.com/tdrn-org/adjudex-mcp/internal/store"
	"github.com/tdrn-org/adjudex-mcp/internal/web"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-httpserver"
)

func main() {
	transport := flag.String("transport", "stdio", "Transport mode: stdio, sse, or http")
	addr := flag.String("addr", ":8080", "Listen address (SSE and HTTP modes)")
	dbPath := flag.String("db", "adjudex.db", "Path to SQLite database file (use \"memory\" for in-memory DB)")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "adjudex v0.2.0 — Agent juDy eXchange\n")

	// Initialize database with adjudex schema
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

	// Wire up MCP server with all five stores
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
		fmt.Fprintf(os.Stderr, "REST API + SvelteKit frontend starting on %s...\n", *addr)
		httpSrv, err := httpserver.Listen(ctx, "tcp", *addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: listen: %v\n", err)
			os.Exit(1)
		}
		// SvelteKit SPA frontend at /
		httpSrv.Handle("/", web.Handler())
		// REST API at /api/v1/
		api.Router(httpSrv, storeSet)
		if err := httpSrv.Serve(); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: serve: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "FATAL: unknown transport %q (use stdio, sse, or http)\n", *transport)
		os.Exit(1)
	}
}
