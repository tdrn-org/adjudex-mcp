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

package notification

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/tdrn-org/adjudex-mcp/config"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-notify/webhook"
)

type webookConfig struct {
	cfg *config.NotifyConfig
}

func (c *webookConfig) GetClient() (*http.Client, error) {
	return http.DefaultClient, nil
}

func (c *webookConfig) GetWebhookURL() (*url.URL, error) {
	return c.cfg.WebhookURL.URL, nil
}

func (c *webookConfig) GetTimeout() (time.Duration, error) {
	return 0, nil
}

func newWebhookService(cfg *config.NotifyConfig) (Service, error) {
	payloadFactory, err := webhook.NewPayloadFactory(&webookConfig{cfg: cfg})
	if err != nil {
		return nil, err
	}
	s := &webhookService{
		payloadFactory: payloadFactory,
		alertPayload:   payloadFactory.NewJSONPayload(cfg.AlertPayload, nil),
	}
	return s, nil
}

type webhookService struct {
	payloadFactory *webhook.PayloadFactory
	alertPayload   *webhook.Payload
}

func (s *webhookService) NotifyAlert(ctx context.Context, alert *domain.Alert) {
	err := s.alertPayload.Send(ctx, newAlertParams(alert))
	if err != nil {
		slog.Error("failed to send alert notification", slog.Any("err", err))
	}
}
