FROM golang:alpine AS build
COPY . /app
WORKDIR /app
RUN GOOS=linux go build -o matrix-alertmanager-receiver

FROM alpine
WORKDIR /app
COPY ./config.toml.sample /etc/matrix-alertmanager-receiver.toml
COPY --from=build /app/matrix-alertmanager-receiver /app/matrix-alertmanager-receiver
EXPOSE 9088
CMD ["./matrix-alertmanager-receiver"]
