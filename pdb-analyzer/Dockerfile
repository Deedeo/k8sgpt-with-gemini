# Use official Go image for build
FROM golang:1.21 AS builder

WORKDIR /app

# Copy your code
COPY . .

# Download dependencies
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pdb-analyzer ./main.go

# Use a minimal image for runtime
FROM alpine:3.18

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy the binary from the builder
COPY --from=builder /app/pdb-analyzer .


# Run the analyzer
CMD ["./pdb-analyzer"]
