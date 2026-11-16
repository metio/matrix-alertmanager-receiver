# SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
# SPDX-License-Identifier: GPL-3.0-or-later

FROM docker.io/library/golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o matrix-alertmanager-receiver

FROM gcr.io/distroless/base-debian13:latest@sha256:0e48b9bcd868bce9fa872cee10b52961d9e32677823da4bf68eefb44f38e7b9d
COPY --from=build /app/matrix-alertmanager-receiver /
USER nonroot:nonroot
ENTRYPOINT ["/matrix-alertmanager-receiver"]
