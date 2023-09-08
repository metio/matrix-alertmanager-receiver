/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package handler

import (
	"net/http"
	"strings"
)

type RoomExtractorFunc func(request *http.Request) string

func CreateRoomExtractor(pathPrefix string) RoomExtractorFunc {
	return func(request *http.Request) string {
		// TODO: switch to new ServerMux Path variables once available in Golang 1.22+
		return strings.TrimPrefix(request.URL.Path, pathPrefix)
	}
}
