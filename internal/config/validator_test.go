/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfiguration_Errors(t *testing.T) {
	ctx := t.Context()
	testCases := map[string]struct {
		configuration *Configuration
		hasErrors     bool
	}{
		"empty": {
			configuration: &Configuration{},
			hasErrors:     true,
		},
		"minimal-http": {
			configuration: &Configuration{
				HTTPServer: HTTPServer{
					Port: 12345,
				},
			},
			hasErrors: true,
		},
		"minimal-matrix": {
			configuration: &Configuration{
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "12345",
					AccessToken:   "secret",
				},
			},
			hasErrors: true,
		},
		"minimal-templating": {
			configuration: &Configuration{
				Templating: Templating{
					Firing: "something broke",
				},
			},
			hasErrors: true,
		},
		"minimal-without-errors": {
			configuration: &Configuration{
				HTTPServer: HTTPServer{
					Port: 12345,
				},
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "12345",
					AccessToken:   "secret",
				},
				Templating: Templating{
					Firing: "something broke",
				},
			},
			hasErrors: false,
		},
		"detect-whitespace-only-template": {
			configuration: &Configuration{
				HTTPServer: HTTPServer{
					Port: 12345,
				},
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "12345",
					AccessToken:   "secret",
				},
				Templating: Templating{
					Firing:   " ",
					Resolved: " ",
				},
			},
			hasErrors: true,
		},
		"detect-whitespace-only-homeserver": {
			configuration: &Configuration{
				Matrix: Matrix{
					HomeServerURL: "",
					UserID:        "12345",
					AccessToken:   "secret",
				},
				Templating: Templating{
					Firing: "abc",
				},
			},
			hasErrors: true,
		},
		"detect-whitespace-only-user": {
			configuration: &Configuration{
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "",
					AccessToken:   "secret",
				},
				Templating: Templating{
					Firing: "abc",
				},
			},
			hasErrors: true,
		},
		"detect-whitespace-only-token": {
			configuration: &Configuration{
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "12345",
					AccessToken:   "",
				},
				Templating: Templating{
					Firing: "abc",
				},
			},
			hasErrors: true,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.hasErrors, validateConfiguration(ctx, testCase.configuration))
		})
	}
}

func TestValidateConfiguration_AdjustedValues(t *testing.T) {
	ctx := t.Context()
	testCases := map[string]struct {
		configuration *Configuration
		expected      *Configuration
	}{
		"minimal-without-errors": {
			configuration: &Configuration{
				HTTPServer: HTTPServer{
					Port: 12345,
				},
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "12345",
					AccessToken:   "secret",
				},
				Templating: Templating{
					Firing: "something broke",
				},
			},
			expected: &Configuration{
				HTTPServer: HTTPServer{
					Port:             12345,
					AlertsPathPrefix: "/alerts/",
					MetricsPath:      "/metrics",
					BasicUsername:    "alertmanager",
				},
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "12345",
					AccessToken:   "secret",
				},
				Templating: Templating{
					Firing: "something broke",
				},
			},
		},
		"with-custom-metrics-path": {
			configuration: &Configuration{
				HTTPServer: HTTPServer{
					Port:        12345,
					MetricsPath: "somewhere-metrics",
				},
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "12345",
					AccessToken:   "secret",
				},
				Templating: Templating{
					Firing: "something broke",
				},
			},
			expected: &Configuration{
				HTTPServer: HTTPServer{
					Port:             12345,
					AlertsPathPrefix: "/alerts/",
					MetricsPath:      "/somewhere-metrics",
					BasicUsername:    "alertmanager",
				},
				Matrix: Matrix{
					HomeServerURL: "example.com",
					UserID:        "12345",
					AccessToken:   "secret",
				},
				Templating: Templating{
					Firing: "something broke",
				},
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.NotEqual(t, testCase.expected, testCase.configuration)
			_ = validateConfiguration(ctx, testCase.configuration)
			assert.Equal(t, testCase.expected, testCase.configuration)
		})
	}
}
