FROM golang:1.20-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o proxy main.go

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/proxy .
ENV MOVIES_MIGRATION_PERCENT=0
EXPOSE 8000
ENTRYPOINT ["./proxy"]
