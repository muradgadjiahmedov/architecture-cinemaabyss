FROM golang:1.20-alpine AS build

WORKDIR /app

# 1) Сначала копируем go.mod и go.sum и тянем зависимости
COPY go.mod go.sum ./
RUN go mod download

# 2) Затем копируем исходники
COPY . .

RUN go build -o events main.go

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/events .
ENV PORT=8082
ENV KAFKA_BROKERS=kafka:9092
ENV KAFKA_TOPIC=events
EXPOSE 8082
ENTRYPOINT ["./events"]
