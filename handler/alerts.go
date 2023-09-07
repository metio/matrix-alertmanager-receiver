/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package handler

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sebhoss/matrix-alertmanager-receiver/alertmanager"
	"github.com/sebhoss/matrix-alertmanager-receiver/matrix"
	"log/slog"
	"net/http"
)

var (
	httpRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_http_requests_total",
		Help: "The total number of HTTP requests received at the /alerts endpoint",
	})
	unsupportedMethodTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_unsupported_http_method_total",
		Help: "The total number of HTTP requests using unsupported HTTP methods received at the /alerts endpoint",
	}, []string{"method"})
	invalidPayloadTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_invalid_payload_total",
		Help: "The total number of HTTP requests that contain invalid payload data at the /alerts endpoint",
	})
	alertsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_alerts_total",
		Help: "The total number of alerts processed",
	}, []string{"room"})
)

func AlertsHandler(ctx context.Context, sendingFunc matrix.SendingFunc, templatingFunc alertmanager.TemplatingFunc, roomExtractorFunc RoomExtractorFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		httpRequestsTotal.Inc()
		if request.Method != http.MethodPost {
			unsupportedMethodTotal.WithLabelValues(request.Method).Inc()
			slog.ErrorContext(ctx, "Unsupported HTTP method used", slog.String("method", request.Method))
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		data, err := alertmanager.DecodePayload(request.Body)
		if err != nil {
			invalidPayloadTotal.Inc()
			slog.ErrorContext(ctx, "Received invalid data", slog.Any("error", err))
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		slog.DebugContext(ctx, "Received valid data", slog.String("remote-address", request.RemoteAddr))

		room := roomExtractorFunc(request)
		slog.DebugContext(ctx, "Extracted roomID", slog.String("room", room))

		for _, alert := range data.Alerts {
			alertsTotal.WithLabelValues(room).Inc()
			if message := templatingFunc(alert, data); message != "" {
				slog.DebugContext(ctx, "Created message", slog.String("html", message))
				sendingFunc(message, room)
			}
		}
		writer.WriteHeader(http.StatusOK)
	}
}
