FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /file-service ./cmd

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && adduser -D -g '' appuser

USER appuser
WORKDIR /home/appuser

COPY --from=builder /file-service /usr/local/bin/file-service

EXPOSE 8082

ENTRYPOINT ["/usr/local/bin/file-service"]
