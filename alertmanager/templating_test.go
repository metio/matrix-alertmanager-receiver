/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package alertmanager

import (
	amtemplate "github.com/prometheus/alertmanager/template"
	"github.com/sebhoss/matrix-alertmanager-receiver/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExternalURL(t *testing.T) {
	testCases := map[string]struct {
		original string
		mapping  map[string]string
		expected string
	}{
		"single-mapped": {
			original: "alertmanager:9093",
			mapping: map[string]string{
				"alertmanager:9093": "https://alertmanager.example.com",
			},
			expected: "https://alertmanager.example.com",
		},
		"multi-mapped": {
			original: "alertmanager:9093",
			mapping: map[string]string{
				"alertmanager:9093": "https://alertmanager.example.com",
				"alerts:12345":      "https://alertmanager.example.com",
			},
			expected: "https://alertmanager.example.com",
		},
		"single-not-mapped": {
			original: "alerts:12345",
			mapping: map[string]string{
				"alertmanager:9093": "https://alertmanager.example.com",
			},
			expected: "alerts:12345",
		},
		"empty-mapping-data": {
			original: "alerts:12345",
			mapping:  map[string]string{},
			expected: "alerts:12345",
		},
		"empty-data": {
			original: "",
			mapping:  map[string]string{},
			expected: "",
		},
		"single-original": {
			original: "",
			mapping: map[string]string{
				"alertmanager:9093": "https://alertmanager.example.com",
			},
			expected: "",
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, externalURL(testCase.original, testCase.mapping))
		})
	}
}

func TestSilenceURL(t *testing.T) {
	testCases := map[string]struct {
		alert       amtemplate.Alert
		externalURL string
		expected    string
	}{
		"no-labels": {
			alert:       amtemplate.Alert{},
			externalURL: "example.com/",
			expected:    "example.com/#/silences/new",
		},
		"no-slash": {
			alert:       amtemplate.Alert{},
			externalURL: "example.com",
			expected:    "example.com/#/silences/new",
		},
		"one-label": {
			alert: amtemplate.Alert{
				Labels: map[string]string{
					"something": "value",
				},
			},
			externalURL: "example.com",
			expected:    "example.com/#/silences/new?filter=%7Bsomething%3D%22value%22%7D",
		},
		"multiple-labels": {
			alert: amtemplate.Alert{
				Labels: map[string]string{
					"something": "value",
					"else":      "other",
					"more":      "labels",
				},
			},
			externalURL: "example.com",
			expected:    "example.com/#/silences/new?filter=%7Belse%3D%22other%22%2C%20more%3D%22labels%22%2C%20something%3D%22value%22%7D",
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, silenceURL(testCase.alert, testCase.externalURL))
		})
	}
}

func TestComputedValues(t *testing.T) {
	testCases := map[string]struct {
		alert    amtemplate.Alert
		values   []config.ComputedValue
		expected map[string]string
	}{
		"no-values": {
			alert:    amtemplate.Alert{},
			values:   []config.ComputedValue{},
			expected: map[string]string{},
		},
		"no-matcher": {
			alert: amtemplate.Alert{},
			values: []config.ComputedValue{
				{
					Values: map[string]string{
						"key": "value",
					},
				},
			},
			expected: map[string]string{
				"key": "value",
			},
		},
		"status-matcher-miss": {
			alert: amtemplate.Alert{
				Status: "firing",
			},
			values: []config.ComputedValue{
				{
					Values: map[string]string{
						"key": "value",
					},
					StatusMatcher: "resolved",
				},
			},
			expected: map[string]string{},
		},
		"status-matcher-hit": {
			alert: amtemplate.Alert{
				Status: "firing",
			},
			values: []config.ComputedValue{
				{
					Values: map[string]string{
						"key": "value",
					},
					StatusMatcher: "firing",
				},
			},
			expected: map[string]string{
				"key": "value",
			},
		},
		"label-matcher-miss": {
			alert: amtemplate.Alert{
				Labels: map[string]string{
					"label": "value",
				},
			},
			values: []config.ComputedValue{
				{
					Values: map[string]string{
						"key": "value",
					},
					LabelMatcher: map[string]string{
						"label": "different",
					},
				},
			},
			expected: map[string]string{},
		},
		"label-matcher-hit": {
			alert: amtemplate.Alert{
				Labels: map[string]string{
					"label": "same",
				},
			},
			values: []config.ComputedValue{
				{
					Values: map[string]string{
						"key": "value",
					},
					LabelMatcher: map[string]string{
						"label": "same",
					},
				},
			},
			expected: map[string]string{
				"key": "value",
			},
		},
		"annotation-matcher-miss": {
			alert: amtemplate.Alert{
				Annotations: map[string]string{
					"annotation": "value",
				},
			},
			values: []config.ComputedValue{
				{
					Values: map[string]string{
						"key": "value",
					},
					AnnotationMatcher: map[string]string{
						"annotation": "different",
					},
				},
			},
			expected: map[string]string{},
		},
		"annotation-matcher-hit": {
			alert: amtemplate.Alert{
				Annotations: map[string]string{
					"annotation": "same",
				},
			},
			values: []config.ComputedValue{
				{
					Values: map[string]string{
						"key": "value",
					},
					AnnotationMatcher: map[string]string{
						"annotation": "same",
					},
				},
			},
			expected: map[string]string{
				"key": "value",
			},
		},
		"multiple-matcher-hit": {
			alert: amtemplate.Alert{
				Status: "firing",
				Labels: map[string]string{
					"label": "same",
				},
				Annotations: map[string]string{
					"annotation": "same",
				},
			},
			values: []config.ComputedValue{
				{
					Values: map[string]string{
						"key": "value",
					},
					StatusMatcher: "firing",
					LabelMatcher: map[string]string{
						"label": "same",
					},
					AnnotationMatcher: map[string]string{
						"annotation": "same",
					},
				},
			},
			expected: map[string]string{
				"key": "value",
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, computeValues(testCase.alert, testCase.values))
		})
	}
}
