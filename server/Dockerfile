# Use the official Golang alpine image
FROM golang:1.21-alpine

WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy your Go code and build the binary
COPY . .
RUN go build -o main .

# Expose your API port
EXPOSE 8080

# Run the compiled binary
CMD ["./main"]