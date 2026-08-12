# --- Stage 1: Builder ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Point specifically to the cmd/api directory containing main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

# --- Stage 2: Final Runtime ---
FROM alpine:latest

# git and docker-cli are what the deploy webhook shells out to: it pulls the
# repo and rebuilds this image against the host docker socket.
RUN apk --no-cache add ca-certificates git docker-cli

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/server .
COPY --from=builder /app/scripts ./scripts

RUN chmod +x ./scripts/*.sh

EXPOSE 8000

CMD ["./server"]
