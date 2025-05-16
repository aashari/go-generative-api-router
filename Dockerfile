FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o generative-api-router ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/generative-api-router .
COPY credentials.json .
COPY models.json .
EXPOSE 8082
CMD ["./generative-api-router"] 