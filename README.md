<!--
SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
SPDX-License-Identifier: GPL-3.0-or-later
 -->

# matrix-alertmanager-receiver

[Alertmanager client](https://prometheus.io/docs/alerting/latest/clients/) that forwards alerts to a [Matrix](https://matrix.org/) room. This is a fork of https://git.sr.ht/~fnux/matrix-alertmanager-receiver with the following changes:

- Add templating mechanism for alerts based on Golang's [html/template](https://pkg.go.dev/html/template)
- Allow arbitrary rooms as receivers with optional pretty URLs
- Mapping of `ExternalURL` values for misconfigured Alertmanager instances
- Computation of `SilenceURL` and arbitrary other values.
- Replace TOML with YAML format
- Add Prometheus metrics for received alerts, sent notifications, and templating failures
- Use the [slog](https://pkg.go.dev/log/slog) package for structured logging

## Usage

Configure your Alertmanager(s) to use this service as a webhook receiver like this:

```yaml
receivers:
  - name: matrix
    webhook_configs:
      - url: "http://example.com:<port>/<alerts-path-prefix>/{roomID}"
```

The values for `<port>` and `<alerts-path-prefix>` are configuration options of this service and need to match whatever you wrote into your Alertmanager configuration. The value for `{roomID}` must be a valid Matrix room ID or a pre-defined pretty URL (see below). The following snippet shows the same configuration with all options specified:

```yaml
receivers:
  - name: some-room
    webhook_configs:
      - url: "http://example.com:12345/alerts/!PFFZ6G9E07n2tnbiUD:matrix.example.com"
  - name: other-room
    webhook_configs:
      - url: "http://example.com:12345/alerts/!HJFZ28f4jKJfmaHLEk:matrix.example.com"
```

Note that you can use the `matrix.room-mapping` configuration option to expose 'pretty' URLs and hide those Matrix room IDs from your Alertmanager configuration:

```yaml
receivers:
  - name: some-room
    webhook_configs:
      - url: "http://example.com:12345/alerts/pager"
  - name: other-room
    webhook_configs:
      - url: "http://example.com:12345/alerts/ticket"
```

## CLI Arguments

This service is a single binary which provides two CLI arguments:

- `--config-path`: Specify the path to the configuration file to use.
- `--log-level`: Specify the log level to use. Possible values are error, warn, debug, info. Defaults to info.

## Configuration

```yaml
# configuration of the HTTP server
http:
  address: 127.0.0.1          # bind address for this service. Can be left unspecified to bind on all interfaces
  port: 12345                 # port used by this service
  alerts-path-prefix: /alerts # URL path for the webhook receiver called by an Alertmanager. Defaults to /alerts
  metrics-path: /metrics      # URL path to collect metrics. Defaults to /metrics
  metrics-enabled: true       # Whether to enable metrics or not. Defaults to false

# configuration for the Matrix connection
matrix:
  homeserver-url: https://matrix.example.com        # FQDN of the homeserver
  user-id: "@user:matrix.example.com"               # ID of the user used by this service
  access-token: secret                              # Access token for the user ID
  # define short names for Matrix room ID
  room-mapping:
    simple-name: "!qohfwef7qwerf:example.com"

# configuration of the templating features
templating:
  # mapping of ExternalURL values
  external-url-mapping:
    # key is the original value taken from the Alertmanager payload
    # value is the mapped value which will be available as '.ExternalURL' in templates
    "http://alertmanager:9093": https://alertmanager.example.com

  # computation of arbitrary values based on matching alert annotations, labels, or status
  # values will be evaluated top to bottom, last entry wins
  computed-values:
    - values: # always set 'color' to 'yellow'
        color: yellow
    - values: # set 'color' to 'orange' when alert label 'severity' is 'warning'
        color: orange
      when-matching-labels:
        severity: warning
    - values: # set 'color' to 'red' when alert label 'severity' is 'critical'
        color: red
      when-matching-labels:
        severity: critical
    - values: # set 'color' to 'green' when alert status is 'resolved'
        color: green
      when-matching-status: resolved

  # template for alerts in status 'firing'
  firing-template: '
    <p>
      <strong><font color="{{ .ComputedValues.color }}">{{ .Alert.Status | ToUpper }}</font></strong>
      {{ if .Alert.Labels.name }}
        {{ .Alert.Labels.name }}
      {{ else if .Alert.Labels.alertname }}
        {{ .Alert.Labels.alertname }}
      {{ end }}
      >>
      {{ if .Alert.Labels.severity }}
        {{ .Alert.Labels.severity | ToUpper }}: 
      {{ end }}
      {{ if .Alert.Annotations.description }}
        {{ .Alert.Annotations.description }}
      {{ else if .Alert.Annotations.summary }}
        {{ .Alert.Annotations.summary }}
      {{ end }}
      >>
      {{ if .Alert.Annotations.runbook }}
        <a href="{{ .Alert.Annotations.runbook }}">Runbook</a> | 
      {{ end }}
      {{ if .Alert.Annotations.dashboard }}
        <a href="{{ .Alert.Annotations.dashboard }}">Dashboard</a> | 
      {{ end }}
      <a href="{{ .SilenceURL }}">Silence</a>
    </p>'

  # template for alerts in status 'resolved', if not specified will use the firing-template
  resolved-template: '
    <strong><font color="{{ .ComputedValues.color }}">{{ .Alert.Status | ToUpper }}</font></strong>{{ .Alert.Labels.name }}'
```

### Templating

Template are written using Golang's [html/template](https://pkg.go.dev/html/template) feature. The following template values are available:

- `Alert`: The [alert data](https://prometheus.io/docs/alerting/latest/notifications/#alert) as provided by the Alertmanager.
- `GroupLabels`: The GroupLabels value of the original [payload](https://prometheus.io/docs/alerting/latest/notifications/#data) sent by the Alertmanager.
- `CommonLabels`: The CommonLabels value of the original [payload](https://prometheus.io/docs/alerting/latest/notifications/#data) sent by the Alertmanager.
- `CommonAnnotations`: The CommonAnnotations value of the original [payload](https://prometheus.io/docs/alerting/latest/notifications/#data) sent by the Alertmanager.
- `ExternalURL`: The ExternalURL value of the original [payload](https://prometheus.io/docs/alerting/latest/notifications/#data) sent by the Alertmanager mapped by the mapping section in the configuration file. If no entry exists in the map, the original value will be available as-is in the template.
- `SilenceURL`: The calculated URL to silence an alert. This should be used like this `<a href="{{ .SilenceURL }}">Silence</a>` or similar.
- `ComputedValues`: Map of computed values defined in the configuration file.

#### ExternalURL

The `ExternalURL` as sent by an Alertmanager contains the backlink to the Alertmanager that sent the notification. In some cases you might want/need to map these values to something that is externally available. Use the `templating.external-url-mapping` configuration key for these cases. Each key is the full original value as sent by an Alertmanager and each value is what you want to use in your templates.

```yaml
templating:
  external-url-mapping:
    "http://alertmanager:9093": https://alertmanager.example.com
    "http://alerts:12345": https://alerts.example.com
```

Using the above configuration, all alerts whose `ExternalURL` original value is `http://alertmanager:9093` will be `https://alertmanager.example.com` and `http://alerts:12345` will be mapped to `https://alerts.example.com`.

#### Computed Values

You can make additional arbitrary values available in your templates by using the `templating.computed-values` key. There are several ways to configure when these values are available in your template and which value they have.

```yaml
templating:
  computed-values:
    - values:
        color: yellow
        something: value
        another: yesplease
```

The above configuration adds the `ComputedValues.color`, `ComputedValues.something`, and `ComputedValues.another` with their respective values to your template.

```yaml
templating:
  computed-values:
    - values:
        color: yellow
        something: value
        another: yesplease
      when-matching-labels:
        severity: warning
        thanos: global
```

The above configuration adds the same key/values only if an alert contains the label `severity` with value `warning` and another label called `thanos` with value `global`.

```yaml
templating:
  computed-values:
    - values:
        color: yellow
        something: value
        another: yesplease
      when-matching-annotations:
        prometheus: production
```

The above configuration adds the same key/values only if an alert contains the annotation `prometheus` with value `production`.

```yaml
templating:
  computed-values:
    - values:
        color: yellow
        something: value
        another: yesplease
      when-matching-status: resolved
```

The above configuration adds the key/values only if an alert has the status `resolved`.

```yaml
templating:
  computed-values:
    - values:
        color: yellow
        something: value
        another: yesplease
      when-matching-labels:
        severity: warning
        thanos: global
      when-matching-annotations:
        prometheus: production
      when-matching-status: resolved
```

The above configuration matches on labels, annotations, and alert status. Only if all specified matchers evaluate to true, the values will be added to your template.

#### Functions

Besides the functions available in plain Golang templates, the following functions can be used in all templates:

- `ToUpper`: Calls the [strings.ToUpper](https://pkg.go.dev/strings#ToUpper) function.
- `ToLower`: Calls the [strings.ToLower](https://pkg.go.dev/strings#ToLower) function.

Please open a ticket in case you need additional functions from the Golang SDK.

## Metrics

The following metrics are available at the `/metrics` endpoint of this service in a Prometheus compatible format:

```
# The total number of HTTP requests received at the /alerts endpoint
matrix_alertmanager_receiver_http_requests_total

# The total number of HTTP requests using unsupported HTTP methods received at the /alerts endpoint
matrix_alertmanager_receiver_unsupported_http_method_total

# The total number of HTTP requests that contain invalid payload data at the /alerts endpoint
matrix_alertmanager_receiver_invalid_payload_total

# The total number of alerts processed
matrix_alertmanager_receiver_alerts_total

# The total number of successful templating operations
matrix_alertmanager_receiver_templating_success_total

# The total number of failed templating operations
matrix_alertmanager_receiver_templating_failure_total

# The total number of failed join room operations
matrix_alertmanager_receiver_join_room_failure_total

# The total number of successful join room operations
matrix_alertmanager_receiver_join_room_success_total

# The total number of failed send operations
matrix_alertmanager_receiver_send_failure_total

# The total number of successful send operations
matrix_alertmanager_receiver_send_success_total
```

## Alternatives

It's highly likely that this project does not meet your needs. Here is a list of potential alternatives you might want to consider:

- https://github.com/matrix-org/matrix-hookshot
- https://github.com/jaywink/matrix-alertmanager
- https://github.com/dkess/alertmanager_matrix
- https://gitlab.com/albalitz/alertmanager-matrix
- https://github.com/silkeh/alertmanager_matrix
- https://github.com/metalmatze/alertmanager-bot

## Building

In order to build this project, make sure to install at least Golang 1.21 and run the following command:

```shell
$ CGO_ENABLED=0 go build -o matrix-alertmanager-receiver
```
