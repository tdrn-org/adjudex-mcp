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

package adjudexmcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/tdrn-org/adjudex-mcp/config"
	"github.com/tdrn-org/adjudex-mcp/internal/adapters/middleware/mcp"
	"github.com/tdrn-org/adjudex-mcp/internal/adapters/middleware/rest"
	"github.com/tdrn-org/adjudex-mcp/internal/data"
	"github.com/tdrn-org/adjudex-mcp/internal/data/model"
	"github.com/tdrn-org/adjudex-mcp/internal/stock"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-database/memory"
	"github.com/tdrn-org/go-database/sqlite"
	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/go-tlsconf/tlsclient"
)

const serverJobTickerSchedule time.Duration = 5 * time.Minute

type Server struct {
	cfg                 *config.Config
	dataStore           *data.Store
	httpServer          *httpserver.Instance
	baseURL             *url.URL
	stockTrackerPool    *stock.TrackerPool
	jobTicker           *time.Ticker
	jobTickerShutdown   chan any
	jobTickerShutdownWG sync.WaitGroup
	jobs                []jobRunner
	logger              *slog.Logger
}

func StartServer(ctx context.Context, cfg *config.Config) (*Server, error) {
	applyLoggingConfig(&cfg.Logging)
	// Setup early logger with configuration address (which may not be the final one).
	// We will reset the logger after listener has been created.
	earlyLogger := slog.With(slog.String("server", cfg.Server.Address))
	s := &Server{
		cfg:    cfg,
		logger: earlyLogger,
	}
	startFuncs := []func(context.Context, *config.Config) error{
		s.startStore,
		s.startHttpServer,
		s.startStockTrackerPool,
		s.startRestAPI,
		s.startMCPHandler,
		s.startJobTicker,
	}
	for _, startFunc := range startFuncs {
		err := startFunc(ctx, cfg)
		if err != nil {
			defer s.Close()
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("serving HTTP requests...")
	err := s.httpServer.Serve()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownFuncs := []func(context.Context) error{
		s.shutdownJobTicker,
		s.shutdownHttpServer,
	}
	shutdownErrs := make([]error, 0, len(shutdownFuncs))
	for _, shutdownFunc := range shutdownFuncs {
		shutdownErrs = append(shutdownErrs, shutdownFunc(ctx))
	}
	return errors.Join(shutdownErrs...)
}

func (s *Server) Close() error {
	closeFuncs := []func() error{
		s.closeHttpServer,
		s.closeStore,
	}
	closeErrs := make([]error, 0, len(closeFuncs))
	for _, closeFunc := range closeFuncs {
		closeErrs = append(closeErrs, closeFunc())
	}
	return errors.Join(closeErrs...)
}

func (s *Server) Ping(ctx context.Context) error {
	if s.httpServer == nil {
		return fmt.Errorf("server not started")
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsclient.GetConfig(),
		},
	}
	pingURL := s.httpServer.BaseURL().JoinPath(rest.PathPing).String()
	rsp, err := client.Get(pingURL)
	if err != nil {
		return fmt.Errorf("failed to access URL: '%s' (cause: %w)", pingURL, err)
	}
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to ping URL: '%s' (status: %s)", pingURL, rsp.Status)
	}
	return nil
}

func (s *Server) startStore(ctx context.Context, cfg *config.Config) error {
	s.logger.Info("starting data store...", slog.String("type", string(cfg.Store.DatabaseType)))
	var databaseConfig database.Config
	switch cfg.Store.DatabaseType {
	case config.DatabaseType(memory.Type):
		databaseConfig = memory.NewConfig(model.SqliteSchemaScriptOption)
	case config.DatabaseType(sqlite.Type):
		databaseConfig = sqlite.NewConfig(cfg.Store.SQLiteConfig.File, sqlite.ModeRWC, model.SqliteSchemaScriptOption)
	default:
		return fmt.Errorf("unrecognized store type '%s'", cfg.Store.DatabaseType)
	}
	driver, err := database.Open(databaseConfig)
	if err != nil {
		return err
	}
	_, _, err = driver.UpdateSchema(ctx)
	if err != nil {
		return errors.Join(err, driver.Close())
	}
	s.dataStore = data.NewStore(databaseConfig.RedactedDSN(), driver)
	return nil
}

func (s *Server) closeStore() error {
	if s.dataStore == nil {
		return nil
	}
	s.logger.Info("closing data store")
	return s.dataStore.Close()
}

func (s *Server) startHttpServer(ctx context.Context, cfg *config.Config) error {
	s.logger.Info("starting HTTP server...", slog.String("address", cfg.Server.Address))
	httpServerOptions := httpServerOptions(&cfg.Server)
	httpServer, err := httpserver.Listen(ctx, "tcp", cfg.Server.Address, httpServerOptions...)
	if err != nil {
		return err
	}
	s.httpServer = httpServer
	if cfg.Server.PublicURL.URL != nil {
		s.baseURL = cfg.Server.PublicURL.URL
	} else {
		s.baseURL = httpServer.BaseURL()
	}
	// Replace early logger by one attributed with actual URL
	s.logger = slog.With(slog.String("baseURL", s.baseURL.String()))
	return nil
}

func (s *Server) shutdownHttpServer(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Info("shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) closeHttpServer() error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Info("closing HTTP server...")
	return s.httpServer.Close()
}

func (s *Server) startStockTrackerPool(_ context.Context, cfg *config.Config) error {
	runtime := &serverRuntime{server: s}
	stockTrackerPool, err := stock.NewTrackerPool(runtime, &cfg.QuoteTracker)
	if err != nil {
		return err
	}
	s.stockTrackerPool = stockTrackerPool
	return nil
}

func (s *Server) startRestAPI(_ context.Context, _ *config.Config) error {
	runtime := &serverRuntime{server: s}
	rest.NewAPI(runtime).Mount(s.httpServer)
	return nil
}

func (s *Server) startMCPHandler(_ context.Context, _ *config.Config) error {
	runtime := &serverRuntime{server: s}
	s.httpServer.Handle("/mcp", mcp.NewHandler(runtime))
	return nil
}

func (s *Server) startJobTicker(_ context.Context, cfg *config.Config) error {
	schedule := serverJobTickerSchedule
	s.logger.Info("starting job ticker...", slog.String("schedule", schedule.String()))
	s.jobTicker = time.NewTicker(schedule)
	s.jobTickerShutdown = make(chan any)
	s.jobs = append(s.jobs, s.stockTrackerPool)
	s.jobTickerShutdownWG.Go(func() {
		for stopped := false; !stopped; {
			select {
			case <-s.jobTickerShutdown:
				stopped = true
			case <-s.jobTicker.C:
				s.runJobs()
			}
		}
		s.logger.Info("job ticker stopped")
	})
	return nil
}

func (s *Server) shutdownJobTicker(_ context.Context) error {
	s.logger.Info("shutting down job ticker...")
	s.jobTicker.Stop()
	s.jobTickerShutdown <- true
	s.jobTickerShutdownWG.Wait()
	return nil
}

type serverRuntime struct {
	server *Server
}

func (runtime *serverRuntime) BaseURL() *url.URL {
	return runtime.server.baseURL
}

func (runtime *serverRuntime) Logger() *slog.Logger {
	return runtime.server.logger
}

func (runtime *serverRuntime) DataStore() *data.Store {
	return runtime.server.dataStore
}

func (runtime *serverRuntime) Ping(ctx context.Context) error {
	err := runtime.server.dataStore.Ping(ctx)
	if err != nil {
		runtime.server.logger.Warn("data store ping failure", slog.Any("err", err))
		return err
	}
	return nil
}
