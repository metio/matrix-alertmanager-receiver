/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"context"
	"log/slog"
	"strings"
)

func validateConfiguration(ctx context.Context, configuration *Configuration) bool {
	hasValidationErrors := false

	http := &configuration.HTTPServer
	if http.Port < 1 || 65535 < http.Port {
		slog.ErrorContext(ctx, "Invalid HTTP port specified", slog.Int("port", http.Port))
		hasValidationErrors = true
	}
	if strings.TrimSpace(http.AlertsPathPrefix) == "" {
		http.AlertsPathPrefix = "/alerts"
	}
	if !strings.HasPrefix(http.AlertsPathPrefix, "/") {
		http.AlertsPathPrefix = "/" + http.AlertsPathPrefix
	}
	if !strings.HasSuffix(http.AlertsPathPrefix, "/") {
		http.AlertsPathPrefix = http.AlertsPathPrefix + "/"
	}
	if strings.TrimSpace(http.MetricsPath) == "" {
		http.MetricsPath = "/metrics"
	}
	if !strings.HasPrefix(http.MetricsPath, "/") {
		http.AlertsPathPrefix = "/" + http.MetricsPath
	}
	if strings.TrimSpace(http.BasicUsername) == "" {
		http.BasicUsername = "alertmanager"
	}

	matrix := &configuration.Matrix
	if strings.TrimSpace(matrix.HomeServerURL) == "" {
		slog.ErrorContext(ctx, "No homeserver URL is set")
		hasValidationErrors = true
	}
	if strings.TrimSpace(matrix.UserID) == "" {
		slog.ErrorContext(ctx, "No user ID is set")
		hasValidationErrors = true
	}
	if strings.TrimSpace(matrix.AccessToken) == "" {
		slog.ErrorContext(ctx, "No access token is set")
		hasValidationErrors = true
	}
	for key, value := range matrix.RoomMapping {
		if strings.TrimSpace(value) == "" {
			slog.ErrorContext(ctx, "Empty room mapping value detected", slog.String("key", key))
			hasValidationErrors = true
		}
	}

	templating := configuration.Templating
	if strings.TrimSpace(templating.Firing) == "" {
		slog.ErrorContext(ctx, "No template for firing alerts defined")
		hasValidationErrors = true
	}

	return hasValidationErrors
}
