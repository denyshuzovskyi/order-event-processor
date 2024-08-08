FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o order-event-processor ./cmd/api-server/main.go

EXPOSE 8080

CMD ["./order-event-processor"]
