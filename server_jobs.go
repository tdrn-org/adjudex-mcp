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
	"fmt"
	"time"
)

type jobFunc func(ctx context.Context)

func (s *Server) runJobs() {
	s.logger.Info("running jobs...")
	ctx := context.Background()
	for _, job := range s.jobs {
		job(ctx)
	}
}

func (s *Server) evalAlerts(ctx context.Context) {
	s.logger.Info("evaluating alerts...")

	alerts, err := s.dataStore.ListArmedAlerts(ctx)
	if err != nil {
		s.logger.Error("failed to list armed alerts", "err", err)
		return
	}

	for _, alert := range alerts {
		quote, err := s.dataStore.GetLatestQuote(ctx, alert.Symbol)
		if err != nil {
			s.logger.Warn("failed to get latest quote for alert", "symbol", alert.Symbol, "err", err)
			continue
		}
		if alert.Evaluate(quote.Price) {
			now := time.Now()
			alert.Fire(now, fmt.Sprintf("%s: %.2f", alert.Condition, quote.Price))
			if err := s.dataStore.UpdateAlert(ctx, &alert); err != nil {
				s.logger.Error("failed to update triggered alert", "alert", alert.Name, "err", err)
			}
			s.logger.Warn("alert triggered!", "alert", alert.Name, "symbol", alert.Symbol, "price", quote.Price)
		}
	}
}
