# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies for sqlite
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the app
RUN go build -o server main.go

# Run Stage
FROM alpine:latest

WORKDIR /app

# Copy binary and static assets
COPY --from=builder /app/server .
COPY --from=builder /app/web ./web
COPY --from=builder /app/.env.example ./.env

EXPOSE 8081

CMD ["./server"]
