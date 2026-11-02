# SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
# SPDX-License-Identifier: GPL-3.0-or-later

FROM docker.io/library/golang:1.27-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o matrix-alertmanager-receiver

FROM gcr.io/distroless/base-debian13:latest@sha256:075a33d9c49026ca7ee72738b4c4a1caea55d0f44227356e46a611a211050e20
COPY --from=build /app/matrix-alertmanager-receiver /
USER nonroot:nonroot
ENTRYPOINT ["/matrix-alertmanager-receiver"]
