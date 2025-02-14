/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package handler

import (
	"net/http"
)

type AuthorizerFunc func(request *http.Request) bool

func CreateAlwaysAllowedAuthorizer() AuthorizerFunc {
	return func(request *http.Request) bool {
		return true
	}
}

func CreateBasicAuthAuthorizer(username string, password string) AuthorizerFunc {
	return func(request *http.Request) bool {
		basicUsername, basicPassword, ok := request.BasicAuth()
		return ok && basicUsername == username && basicPassword == password
	}
}
