FROM golang:1.24.2

WORKDIR /messageApp

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/main.go

EXPOSE 8081

CMD ["./main"]
