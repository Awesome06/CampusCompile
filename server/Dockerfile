# --- Stage 1: The Builder ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies first (saves time on rebuilds)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your source code
COPY . .

# Compile a microscopic, statically-linked binary.
# We strip debugging info (-s -w) to drastically reduce the file size.
RUN CGO_ENABLED=0 GOOS=linux GOMAXPROCS=1 go build -p 1 -ldflags="-s -w" -o api ./cmd/api

# --- Stage 2: The Runner ---
# We switch to a bare-bones Alpine Linux image that has ZERO Go tooling inside it
FROM alpine:latest

WORKDIR /app

# Install security certificates so Go can make secure HTTPS calls (like to AWS S3/MinIO)
RUN apk --no-cache add ca-certificates tzdata

# Copy ONLY the compiled binary from the builder stage
COPY --from=builder /app/api .

# Expose your API port
EXPOSE 8080

# Execute the binary
CMD ["./api"]