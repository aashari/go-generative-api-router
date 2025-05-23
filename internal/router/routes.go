package router

import (
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/handlers"
	"github.com/aashari/go-generative-api-router/internal/monitoring"
	httpSwagger "github.com/swaggo/http-swagger"
)

// SetupRoutes configures all routes for the application
func SetupRoutes(apiHandlers *handlers.APIHandlers) http.Handler {
	mux := http.NewServeMux()

	// Register API handlers
	mux.HandleFunc("/health", apiHandlers.HealthHandler)
	mux.HandleFunc("/v1/chat/completions", apiHandlers.ChatCompletionsHandler)
	mux.HandleFunc("/v1/models", apiHandlers.ModelsHandler)

	// Add metrics endpoint
	mux.HandleFunc("/metrics", monitoring.MetricsHandler)

	// Add pprof endpoints for performance profiling
	monitoring.SetupPprofRoutes(mux)

	// Serve Swagger UI with proper configuration
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The URL pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Wrap with metrics middleware
	return monitoring.MetricsMiddleware(mux)
}
