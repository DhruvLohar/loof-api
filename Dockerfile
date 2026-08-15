# --- Stage 1: Builder ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Point specifically to the cmd/api directory containing main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

# deploy.sh runs migrations before swapping containers, so goose ships in the
# runtime image. Pinned so a deploy can't pick up a breaking release on its own.
#
# The no_* tags drop every driver except postgres. Stock goose blank-imports all
# of them, which drags in clickhouse, ydb, vertica, mssql and a cgo-free sqlite
# -- hundreds of packages we never call, and enough build output to fill a small
# EC2 root volume mid-build.
RUN CGO_ENABLED=0 GOOS=linux go install \
	-tags='no_clickhouse no_mssql no_mysql no_sqlite3 no_turso no_vertica no_ydb' \
	github.com/pressly/goose/v3/cmd/goose@v3.27.3

# --- Stage 2: Final Runtime ---
FROM alpine:latest

# git and docker-cli are what the deploy webhook shells out to: it pulls the
# repo and rebuilds this image against the host docker socket. tzdata backs the
# TimeZone in the database DSN, which alpine has no zoneinfo for otherwise.
RUN apk --no-cache add ca-certificates git docker-cli tzdata

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/server .
COPY --from=builder /app/scripts ./scripts
COPY --from=builder /go/bin/goose /usr/local/bin/goose

RUN chmod +x ./scripts/*.sh

EXPOSE 8000

CMD ["./server"]
