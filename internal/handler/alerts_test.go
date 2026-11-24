/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	amtemplate "github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/assert"
)

func TestAlertsHandlerSendResults(t *testing.T) {
	testCases := map[string]struct {
		alerts         []string
		failingSends   []string
		brokenTemplate []string
		wantStatus     int
		wantAttempts   []string
	}{
		"all-sends-succeed": {
			alerts:       []string{"first", "second"},
			wantStatus:   http.StatusOK,
			wantAttempts: []string{"first", "second"},
		},
		"first-of-two-sends-fails": {
			alerts:       []string{"first", "second"},
			failingSends: []string{"first"},
			wantStatus:   http.StatusBadGateway,
			wantAttempts: []string{"first", "second"},
		},
		"all-sends-fail": {
			alerts:       []string{"first", "second"},
			failingSends: []string{"first", "second"},
			wantStatus:   http.StatusBadGateway,
			wantAttempts: []string{"first", "second"},
		},
		"templating-error-is-not-a-send-failure": {
			alerts:         []string{"first", "second"},
			brokenTemplate: []string{"first"},
			wantStatus:     http.StatusOK,
			wantAttempts:   []string{"second"},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			var attempts []string
			sendingFunc := func(htmlText string, roomID string) error {
				attempts = append(attempts, htmlText)
				for _, failing := range testCase.failingSends {
					if failing == htmlText {
						return errors.New("homeserver unavailable")
					}
				}
				return nil
			}
			templatingFunc := func(alert amtemplate.Alert, _ *amtemplate.Data) (string, error) {
				alertName := alert.Labels["alertname"]
				for _, broken := range testCase.brokenTemplate {
					if broken == alertName {
						return "", errors.New("template failed")
					}
				}
				return alertName, nil
			}
			handler := AlertsHandler(
				context.Background(),
				sendingFunc,
				templatingFunc,
				func(*http.Request) string { return "!room:example.org" },
				CreateAlwaysAllowedAuthorizer(),
			)

			data := amtemplate.Data{}
			for _, alertName := range testCase.alerts {
				data.Alerts = append(data.Alerts, amtemplate.Alert{Labels: amtemplate.KV{"alertname": alertName}})
			}
			payload, err := json.Marshal(data)
			assert.NoError(t, err)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/alerts/room", strings.NewReader(string(payload)))
			handler(recorder, request)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
			assert.Equal(t, testCase.wantAttempts, attempts)
		})
	}
}
