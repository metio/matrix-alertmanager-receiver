/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package alertmanager

import (
	"encoding/json"
	"github.com/prometheus/alertmanager/template"
	"io"
)

func DecodePayload(requestBody io.ReadCloser) (*template.Data, error) {
	payload := template.Data{}
	err := json.NewDecoder(requestBody).Decode(&payload)
	return &payload, err
}
