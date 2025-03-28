/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/metio/matrix-alertmanager-receiver/internal/alertmanager"
	"github.com/metio/matrix-alertmanager-receiver/internal/config"
	"github.com/metio/matrix-alertmanager-receiver/internal/handler"
	"github.com/metio/matrix-alertmanager-receiver/internal/matrix"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

var matrixAlertmanagerReceiverVersion = "development"

func main() {
	var configPath = flag.String("config-path", "", "Path to configuration file")
	var logLevel = flag.String("log-level", "info", "The log level to use (debug, info, warn, error)")
	var version = flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Println(matrixAlertmanagerReceiverVersion)
		os.Exit(0)
	}

	configureLogger(logLevel)
	ctx := context.Background()

	if configPath == nil || *configPath == "" {
		slog.ErrorContext(ctx, "No --config-path parameter specified")
		os.Exit(1)
	}
	slog.InfoContext(ctx, "CLI flags parsed",
		slog.String("config-path", *configPath),
		slog.String("log-level", *logLevel))

	configuration := config.ParseConfiguration(ctx, *configPath)
	if configuration == nil {
		slog.ErrorContext(ctx, "Could not parse configuration")
		os.Exit(1)
	}
	slog.InfoContext(ctx, "Configuration parsed", slog.Any("configuration", configuration.LogValue()))

	sendingFunc := matrix.CreatingSendingFunc(ctx, configuration.Matrix)
	slog.InfoContext(ctx, "Matrix sending function created")

	templatingFunc := alertmanager.CreateTemplatingFunc(ctx, configuration.Templating)
	slog.InfoContext(ctx, "Message templating function created")

	extractorFunc := handler.CreateRoomExtractor(configuration.HTTPServer.AlertsPathPrefix)
	slog.InfoContext(ctx, "Room extracting function created")

	var authorizerFunc handler.AuthorizerFunc
	if configuration.HTTPServer.BasicPassword != "" {
		slog.InfoContext(ctx, "Configuring basic authentication")
		authorizerFunc = handler.CreateBasicAuthAuthorizer(configuration.HTTPServer.BasicUsername, configuration.HTTPServer.BasicPassword)
	} else {
		slog.InfoContext(ctx, "Allowing all incoming requests")
		authorizerFunc = handler.CreateAlwaysAllowedAuthorizer()
	}
	slog.InfoContext(ctx, "Request authorizer function created")

	http.HandleFunc(configuration.HTTPServer.AlertsPathPrefix, handler.AlertsHandler(ctx, sendingFunc, templatingFunc, extractorFunc, authorizerFunc))
	if configuration.HTTPServer.MetricsEnabled {
		slog.InfoContext(ctx, "Enabling metrics endpoint")
		http.Handle(configuration.HTTPServer.MetricsPath, promhttp.Handler())
	}
	slog.InfoContext(ctx, "Handlers configured")

	var listenAddr = fmt.Sprintf("%v:%v", configuration.HTTPServer.Address, configuration.HTTPServer.Port)
	err := http.ListenAndServe(listenAddr, nil)
	if errors.Is(err, http.ErrServerClosed) {
		slog.DebugContext(ctx, "Server closed")
		os.Exit(0)
	} else if err != nil {
		slog.ErrorContext(ctx, "Error while serving", slog.Any("error", err))
		os.Exit(1)
	}
}

func configureLogger(logLevel *string) {
	var level slog.Level
	switch strings.ToLower(*logLevel) {
	case "error":
		level = slog.LevelError
	case "warn":
		level = slog.LevelWarn
	case "debug":
		level = slog.LevelDebug
	case "info":
	default:
		level = slog.LevelInfo
	}
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(logHandler))
}
