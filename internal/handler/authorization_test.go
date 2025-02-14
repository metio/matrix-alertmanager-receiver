/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package handler

import (
	"net/http"
	"reflect"
	"testing"
)

func TestAlwaysAllowedAuthorizer(t *testing.T) {
	tests := map[string]struct {
		authorizer AuthorizerFunc
		want       bool
	}{
		"always-allowed": {
			authorizer: CreateAlwaysAllowedAuthorizer(),
			want:       true,
		},
	}
	for name, testcase := range tests {
		t.Run(name, func(t *testing.T) {
			request := http.Request{}
			if got := testcase.authorizer(&request); !reflect.DeepEqual(got, testcase.want) {
				t.Errorf("got %v, want %v", got, testcase.want)
			}
		})
	}
}

func TestCreateBasicAuthAuthorizer(t *testing.T) {
	tests := map[string]struct {
		authorizer AuthorizerFunc
		username   string
		password   string
		want       bool
	}{
		"correct-credentials": {
			authorizer: CreateBasicAuthAuthorizer("alertmanager", "some-password"),
			username:   "alertmanager",
			password:   "some-password",
			want:       true,
		},
		"wrong-username": {
			authorizer: CreateBasicAuthAuthorizer("alertmanager", "some-password"),
			username:   "someone",
			password:   "some-password",
			want:       false,
		},
		"wrong-password": {
			authorizer: CreateBasicAuthAuthorizer("alertmanager", "some-password"),
			username:   "alertmanager",
			password:   "something",
			want:       false,
		},
		"no-credentials": {
			authorizer: CreateBasicAuthAuthorizer("alertmanager", "some-password"),
			username:   "",
			password:   "",
			want:       false,
		},
	}
	for name, testcase := range tests {
		t.Run(name, func(t *testing.T) {
			request := http.Request{Header: map[string][]string{}}
			if testcase.username != "" && testcase.password != "" {
				request.SetBasicAuth(testcase.username, testcase.password)
			}
			if got := testcase.authorizer(&request); !reflect.DeepEqual(got, testcase.want) {
				t.Errorf("got %v, want %v", got, testcase.want)
			}
		})
	}
}
