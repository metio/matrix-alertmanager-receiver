/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package alertmanager

import (
	"bytes"
	"context"
	"fmt"
	amtemplate "github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	"github.com/sebhoss/matrix-alertmanager-receiver/internal/config"
	"html/template"
	"log/slog"
	"maps"
	"net/url"
	"sort"
	"strings"
)

var (
	templatingSuccessTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_templating_success_total",
		Help: "The total number of successful templating operations",
	})
	templatingFailureTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_templating_failure_total",
		Help: "The total number of failed templating operations",
	})
)

type TemplatingFunc func(alert amtemplate.Alert, data *amtemplate.Data) (string, error)

type templateData struct {
	Alert             amtemplate.Alert
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	SilenceURL        string
	ExternalURL       string
	GeneratorURL      string
	ComputedValues    map[string]string
}

func CreateTemplatingFunc(ctx context.Context, configuration config.Templating) TemplatingFunc {
	slog.DebugContext(ctx, "Creating templating function", slog.Any("configuration", configuration.LogValue()))

	templateFunctions := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
	}

	firing := template.Must(template.New("firing").Funcs(templateFunctions).Parse(configuration.Firing))
	resolvedTemplate := configuration.Resolved
	if resolvedTemplate == "" {
		resolvedTemplate = configuration.Firing
	}
	resolved := template.Must(template.New("resolved").Funcs(templateFunctions).Parse(resolvedTemplate))

	return func(alert amtemplate.Alert, data *amtemplate.Data) (string, error) {
		selectedTemplate := firing
		if alert.Status == string(model.AlertResolved) {
			selectedTemplate = resolved
		}

		externalUrl := maybeMapValue(data.ExternalURL, configuration.ExternalURLMapping)
		slog.DebugContext(ctx, "ExternalURL mapped",
			slog.String("original-url", data.ExternalURL),
			slog.String("mapped-url", externalUrl))

		generatorUrl := maybeMapValue(alert.GeneratorURL, configuration.GeneratorURLMapping)
		slog.DebugContext(ctx, "GeneratorURL mapped",
			slog.String("original-url", data.ExternalURL),
			slog.String("mapped-url", externalUrl))

		silenceUrl := silenceURL(alert, externalUrl)
		slog.DebugContext(ctx, "Silence URL computed", slog.String("silence-url", silenceUrl))

		values := computeValues(alert, configuration.ComputedValues)
		slog.DebugContext(ctx, "Values computed", slog.Any("values", values))

		var output bytes.Buffer
		err := selectedTemplate.Execute(&output, templateData{
			Alert:             alert,
			GroupLabels:       data.GroupLabels,
			CommonLabels:      data.GroupLabels,
			CommonAnnotations: data.GroupLabels,
			SilenceURL:        silenceUrl,
			ExternalURL:       externalUrl,
			GeneratorURL:      generatorUrl,
			ComputedValues:    values,
		})
		if err != nil {
			templatingFailureTotal.Inc()
			slog.ErrorContext(ctx, "Cannot template given data", slog.Any("error", err))
			return "", err
		}
		templatingSuccessTotal.Inc()
		return output.String(), nil
	}
}

func computeValues(alert amtemplate.Alert, values []config.ComputedValue) map[string]string {
	computedValues := make(map[string]string)
	for _, computer := range values {
		if len(computer.Values) == 0 {
			continue
		}

		statusMatches := true
		if computer.StatusMatcher != "" {
			if alert.Status != computer.StatusMatcher {
				statusMatches = false
			}
		}

		labelsMatch := true
		if len(computer.LabelMatcher) > 0 {
			for k, v := range computer.LabelMatcher {
				if labelValue, ok := alert.Labels[k]; !ok || labelValue != v {
					labelsMatch = false
					break
				}
			}
		}

		annotationsMatch := true
		if len(computer.AnnotationMatcher) > 0 {
			for k, v := range computer.AnnotationMatcher {
				if annotationValue, ok := alert.Annotations[k]; !ok || annotationValue != v {
					annotationsMatch = false
					break
				}
			}
		}

		if statusMatches && labelsMatch && annotationsMatch {
			maps.Copy(computedValues, computer.Values)
		}
	}

	return computedValues
}

func maybeMapValue(original string, mapping map[string]string) string {
	if replacement, ok := mapping[original]; ok {
		return replacement
	}
	return original
}

func silenceURL(alert amtemplate.Alert, externalURL string) string {
	if externalURL == "" {
		return ""
	}
	return fmt.Sprintf(`%s/#/silences/new%s`,
		strings.TrimSuffix(externalURL, "/"),
		silenceFilter(alert.Labels))
}

func silenceFilter(labels amtemplate.KV) string {
	if len(labels) == 0 {
		return ""
	}
	var filters []string
	for key, value := range labels {
		filters = append(filters, fmt.Sprintf(`%s="%s"`, key, value))
	}
	sort.SliceStable(filters, func(i, j int) bool {
		return filters[i] < filters[j]
	})
	allFilters := strings.Join(filters, ", ")
	filterValue := strings.ReplaceAll(url.QueryEscape(fmt.Sprintf("{%s}", allFilters)), "+", "%20")
	return fmt.Sprintf(`?filter=%s`, filterValue)
}
