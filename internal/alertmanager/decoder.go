/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package alertmanager

import (
	"encoding/json"
	"io"

	"github.com/prometheus/alertmanager/template"
)

func DecodePayload(requestBody io.ReadCloser) (*template.Data, error) {
	payload := template.Data{}
	err := json.NewDecoder(requestBody).Decode(&payload)
	return &payload, err
}
