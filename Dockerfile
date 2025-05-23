FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .

# Install swag for Swagger generation
# TODO: Fix Swagger type definitions for ChatCompletionRequest and other types
# RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger documentation
# RUN $(go env GOPATH)/bin/swag init -g cmd/server/main.go

# Build the application
RUN GOOS=linux go build -o generative-api-router ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/generative-api-router .
# Copy pre-existing docs directory (already generated)
COPY docs ./docs
COPY credentials.json .
COPY models.json .

# Set environment variable to force Go's standard logger to use UTC time and include microseconds
ENV TZ=UTC

# Ensure logs are sent to stdout/stderr for CloudWatch collection
ENV GLOG_logtostderr=1

# Expose the application port
EXPOSE 8082

# Run the application
CMD ["./generative-api-router"] 