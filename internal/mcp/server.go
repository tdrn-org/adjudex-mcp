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

// Package mcp provides the MCP server setup for adjudex.
// Supports stdio (for local Claude Desktop) and SSE (for remote agents) transports.
package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/mcp/tools"
)

// Server wraps the MCP server with adjudex dependencies.
type Server struct {
	srv    *server.MCPServer
	stores *StoreSet
}

// StoreSet bundles all five store interfaces for injection into tools.
type StoreSet struct {
	Portfolio domain.PortfolioStore
	Quote     domain.QuoteStore
	Alert     domain.AlertStore
	Trade     domain.TradeStore
	Strategy  domain.StrategyStore
}

// New creates an adjudex MCP server with all 21 tools registered.
// Call srv.Serve() with the desired transport (stdio or SSE).
func New(storeSet StoreSet) *Server {
	mcpServer := server.NewMCPServer("adjudex", "0.2.0")

	// Register all 21 MCP tools grouped by domain.
	tools.RegisterPortfolioTools(mcpServer, storeSet.Portfolio, storeSet.Quote)
	tools.RegisterQuoteTools(mcpServer, storeSet.Quote)
	tools.RegisterAlertTools(mcpServer, storeSet.Alert)
	tools.RegisterStrategyTools(mcpServer, storeSet.Strategy, storeSet.Quote, storeSet.Trade)
	tools.RegisterTradeTools(mcpServer, storeSet.Trade)

	return &Server{srv: mcpServer, stores: &storeSet}
}

// ServeStdio starts the MCP server on standard input/output (for Claude Desktop).
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.srv)
}

// ServeSSE starts the MCP server on HTTP SSE transport.
// addr is the listen address, e.g., ":8080".
func (s *Server) ServeSSE(addr string) error {
	sseServer := server.NewSSEServer(s.srv)
	return sseServer.Start(addr)
}
