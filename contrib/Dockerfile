# SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
# SPDX-License-Identifier: GPL-3.0-or-later

FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o matrix-alertmanager-receiver

FROM gcr.io/distroless/base-debian11
COPY --from=build /app/matrix-alertmanager-receiver /
USER nonroot:nonroot
ENTRYPOINT ["/matrix-alertmanager-receiver"]
