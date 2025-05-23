package httpclient

import (
	"net/http"
	"time"
)

// Options holds HTTP client configuration options
type Options struct {
	Timeout    time.Duration
	MaxRetries int
	UserAgent  string
}

// Factory creates configured HTTP clients
type Factory struct {
	defaultOptions Options
}

// NewFactory creates a new HTTP client factory with default options
func NewFactory(defaultOptions Options) *Factory {
	if defaultOptions.Timeout == 0 {
		defaultOptions.Timeout = 60 * time.Second
	}
	if defaultOptions.UserAgent == "" {
		defaultOptions.UserAgent = "GenerativeAPIRouter/1.0"
	}
	
	return &Factory{
		defaultOptions: defaultOptions,
	}
}

// CreateClient creates a new HTTP client with the specified options
func (f *Factory) CreateClient(options Options) (*http.Client, error) {
	// Merge with defaults
	if options.Timeout == 0 {
		options.Timeout = f.defaultOptions.Timeout
	}
	if options.UserAgent == "" {
		options.UserAgent = f.defaultOptions.UserAgent
	}
	
	// Create transport with reasonable defaults
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}
	
	client := &http.Client{
		Timeout:   options.Timeout,
		Transport: transport,
	}
	
	return client, nil
}

// CreateDefaultClient creates a client with default options
func (f *Factory) CreateDefaultClient() (*http.Client, error) {
	return f.CreateClient(Options{})
} 