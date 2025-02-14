/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"log/slog"
)

type Configuration struct {
	HTTPServer HTTPServer `json:"http"`
	Matrix     Matrix     `json:"matrix"`
	Templating Templating `json:"templating"`
}

func (c *Configuration) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("http", c.HTTPServer.LogValue()),
		slog.Any("matrix", c.Matrix.LogValue()),
		slog.Any("templating", c.Templating.LogValue()),
	)
}

type HTTPServer struct {
	Address          string `json:"address"`
	Port             int    `json:"port"`
	AlertsPathPrefix string `json:"alerts-path-prefix"`
	MetricsPath      string `json:"metrics-path"`
	MetricsEnabled   bool   `json:"metrics-enabled"`
	BasicUsername    string `json:"basic-username"`
	BasicPassword    string `json:"basic-password"`
}

func (h *HTTPServer) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("address", h.Address),
		slog.Int("port", h.Port),
		slog.String("alerts-path", h.AlertsPathPrefix),
		slog.String("metrics-path", h.MetricsPath),
		slog.Bool("metrics-enabled", h.MetricsEnabled),
		slog.String("basic-username", h.BasicUsername),
	)
}

type Matrix struct {
	HomeServerURL string            `json:"homeserver-url"`
	UserID        string            `json:"user-id"`
	AccessToken   string            `json:"access-token"`
	RoomMapping   map[string]string `json:"room-mapping"`
}

func (m *Matrix) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("homeserver-url", m.HomeServerURL),
		slog.String("user-id", m.UserID),
		slog.Any("room-mapping", m.RoomMapping),
	)
}

type Templating struct {
	ExternalURLMapping  KeyValue        `json:"external-url-mapping"`
	GeneratorURLMapping KeyValue        `json:"generator-url-mapping"`
	ComputedValues      []ComputedValue `json:"computed-values"`
	Firing              string          `json:"firing-template"`
	Resolved            string          `json:"resolved-template"`
}

func (t *Templating) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("firing-template", t.Firing),
		slog.String("resolved-template", t.Resolved),
		slog.Any("external-url-mapping", t.ExternalURLMapping),
		slog.Any("computed-values", t.ComputedValues),
	)
}

type KeyValue map[string]string

type ComputedValue struct {
	Values            KeyValue `json:"values"`
	LabelMatcher      KeyValue `json:"when-matching-labels"`
	AnnotationMatcher KeyValue `json:"when-matching-annotations"`
	StatusMatcher     string   `json:"when-matching-status"`
}
