FROM golang:1.24-alpine AS builder

# Accept VERSION build argument from CodeBuild
ARG VERSION=unknown
ENV VERSION=${VERSION}

WORKDIR /app
COPY . .

# Install swag for Swagger generation
# TODO: Fix Swagger type definitions for ChatCompletionRequest and other types
# RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger documentation
# RUN $(go env GOPATH)/bin/swag init -g cmd/server/main.go

# Build the application with version information
RUN GOOS=linux go build -ldflags "-X main.version=${VERSION}" -o generative-api-router ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/generative-api-router .
# Copy pre-existing docs directory (already generated)
COPY docs ./docs
# Copy configuration files from configs directory
# Copy credentials.json if it exists, otherwise use the example file
COPY configs/credentials.json* configs/models.json ./configs/
# If credentials.json doesn't exist, use the example
RUN if [ ! -f ./configs/credentials.json ] && [ -f ./configs/credentials.json.example ]; then \
    cp ./configs/credentials.json.example ./configs/credentials.json; \
    fi

# Set environment variable to force Go's standard logger to use UTC time and include microseconds
ENV TZ=UTC

# Configure structured logging for production
ENV LOG_LEVEL=INFO
ENV LOG_FORMAT=json
ENV LOG_OUTPUT=stdout

# Ensure logs are sent to stdout/stderr for CloudWatch collection
ENV GLOG_logtostderr=1

# Expose the application port
EXPOSE 8082

# Run the application
CMD ["./generative-api-router"] 