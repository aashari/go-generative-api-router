// Package api provides the Swagger documentation for the API.
package api

// @title           Generative API Router
// @version         2.0.1
// @description     A Go microservice that proxies OpenAI-compatible API calls to multiple LLM vendors (OpenAI, Gemini) using configurable selection strategies. Provides transparent proxy behavior while maintaining complete OpenAI API compatibility.
// @termsOfService  https://github.com/aashari/go-generative-api-router/blob/main/LICENSE

// @contact.name   API Support
// @contact.url    https://github.com/aashari/go-generative-api-router
// @contact.email  a.ashari1302@gmail.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8082
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the API key value (optional - router manages vendor authentication).
