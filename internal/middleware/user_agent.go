package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/aashari/go-generative-api-router/internal/errors"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/utils"
)

// UserAgentFilterMiddleware filters requests based on User-Agent header
// Only allows requests with User-Agent starting with "BrainyBuddy-API"
// Exceptions: /health, /swagger, /swagger/*, /debug/pprof/*
// When ENVIRONMENT=local, this middleware is disabled
func UserAgentFilterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the environment is local, skip user agent validation if it is
		if strings.ToLower(os.Getenv("ENVIRONMENT")) == "local" {
			next.ServeHTTP(w, r)
			return
		}

		// Define allowed paths that bypass User-Agent filtering
		allowedPaths := []string{
			"/health",
			"/swagger",
			"/swagger/",
			"/debug/pprof/",
		}

		// Check if the request path is in the allowed list
		isAllowed := false
		for _, allowedPath := range allowedPaths {
			if r.URL.Path == allowedPath || strings.HasPrefix(r.URL.Path, allowedPath) {
				isAllowed = true
				break
			}
		}

		// If path is allowed, skip User-Agent validation
		if isAllowed {
			next.ServeHTTP(w, r)
			return
		}

		// Get User-Agent header
		userAgent := r.Header.Get(utils.HeaderUserAgent)

		// Check if User-Agent starts with "BrainyBuddy-API"
		if !strings.HasPrefix(userAgent, utils.UserAgentPrefix) {
			// Log the blocked request
			ctx := logger.WithComponent(r.Context(), "UserAgentMiddleware")
			ctx = logger.WithStage(ctx, "RequestBlocked")
			logger.Warn(ctx, "Request blocked by User-Agent filter",
				"reason", "invalid_user_agent",
				"method", r.Method,
				"path", r.URL.Path,
				"user_agent", userAgent,
				"remote_addr", r.RemoteAddr,
			)

			// Return 403 Forbidden with structured error response
			err := errors.NewAuthorizationError("Access denied: Invalid User-Agent")
			errors.HandleError(w, err, http.StatusForbidden)
			return
		}

		// User-Agent is valid, proceed with the request
		next.ServeHTTP(w, r)
	})
}
