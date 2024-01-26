FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o matrix-alertmanager-receiver

FROM gcr.io/distroless/base-debian11
COPY --from=build /app/matrix-alertmanager-receiver /
USER nonroot:nonroot
ENTRYPOINT ["/matrix-alertmanager-receiver"]
