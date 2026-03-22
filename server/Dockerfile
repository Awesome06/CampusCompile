# --- Stage 1: The Builder ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies first
COPY server/go.mod server/go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY server/ .

# Compile the statically-linked binary.
# RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api ./cmd/api
# The following line is for single core processors / Resource limitations
RUN CGO_ENABLED=0 GOOS=linux GOMAXPROCS=1 go build -p 1 -ldflags="-s -w" -o api ./cmd/api

# --- Stage 2: The Runner ---
FROM alpine:latest

WORKDIR /app

RUN addgroup -S nonroot && adduser -S nonroot -G nonroot
RUN apk --no-cache add ca-certificates tzdata

# Copy the compiled binary
COPY --from=builder /app/api .

# COPY THE SQL FILE SO THE SERVER CAN READ IT ON STARTUP
COPY scripts/ddl.sql ./scripts/ddl.sql 

RUN chown -R nonroot:nonroot /app
USER nonroot

EXPOSE 8080

CMD ["./api"]