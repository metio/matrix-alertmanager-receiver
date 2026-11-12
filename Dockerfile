# SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
# SPDX-License-Identifier: GPL-3.0-or-later

FROM docker.io/library/golang:1.27-alpine@sha256:eb61c3873c97416fff0116553fff65d67a3549f72e11d9758849a689cdd8b523 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o matrix-alertmanager-receiver

FROM gcr.io/distroless/base-debian13:latest@sha256:0e48b9bcd868bce9fa872cee10b52961d9e32677823da4bf68eefb44f38e7b9d
COPY --from=build /app/matrix-alertmanager-receiver /
USER nonroot:nonroot
ENTRYPOINT ["/matrix-alertmanager-receiver"]
