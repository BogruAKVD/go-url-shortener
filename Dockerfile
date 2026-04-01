FROM golang:1.25 AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY config ./config
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /url-shortener ./cmd/server

FROM gcr.io/distroless/static-debian12

COPY --from=build /url-shortener /url-shortener

EXPOSE 8080

ENTRYPOINT ["/url-shortener"]
