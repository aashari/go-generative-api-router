FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN GOOS=linux go build -o generative-api-router ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/generative-api-router .
COPY --from=builder /app/docs ./docs
COPY credentials.json .
COPY models.json .
EXPOSE 8082
CMD ["./generative-api-router"] 