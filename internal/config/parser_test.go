/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfiguration(t *testing.T) {
	testCases := map[string]struct {
		configuration string
		expected      *Configuration
		environ       map[string]string
	}{
		"minimal": {
			configuration: `
http:
  port: 12345
matrix:
  homeserver-url: https://matrix.example.com
  user-id: "@user:matrix.example.com"
  access-token: "${SECRET}"
templating:
  firing-template: "something broke ${UNKNOWN}"
`,
			environ: map[string]string{
				"SECRET": "something",
			},
			expected: &Configuration{
				HTTPServer: HTTPServer{
					Port:             12345,
					AlertsPathPrefix: "/alerts/",
					MetricsPath:      "/metrics",
					BasicUsername:    "alertmanager",
				},
				Matrix: Matrix{
					HomeServerURL: "https://matrix.example.com",
					UserID:        "@user:matrix.example.com",
					AccessToken:   "something",
				},
				Templating: Templating{
					Firing: "something broke ${UNKNOWN}",
				},
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			for key, value := range testCase.environ {
				err := os.Setenv(key, value)
				if err != nil {
					t.Fatal(err)
				}
			}
			tempDir := t.TempDir()
			err := os.WriteFile(tempDir+name, []byte(testCase.configuration), 0666)
			if err != nil {
				t.Fatal(err)
			}
			config := ParseConfiguration(t.Context(), tempDir+name)
			assert.Equal(t, testCase.expected, config)
		})
	}
}
