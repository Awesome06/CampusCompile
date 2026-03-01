# Use the official Golang alpine image
FROM golang:1.25-alpine

WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy your Go code into the container
COPY . .

# 👇 Tell Docker to build the API from the new specific path
RUN go build -o api ./cmd/api

# Expose your API port
EXPOSE 8080

# Run the compiled binary
CMD ["./api"]