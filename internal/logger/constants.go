package logger

// LogStages defines standardized stage names for consistent logging
var LogStages = struct {
	// Request lifecycle
	Request     string
	Processing  string
	Response    string
	Validation  string
	Selection   string
	Preparation string

	// Vendor operations
	VendorRequest  string
	VendorResponse string
	VendorError    string

	// System operations
	Initialization    string
	Configuration     string
	DatabaseOperation string

	// Results
	Success  string
	Error    string
	Retry    string
	Fallback string

	// Middleware stages
	MiddlewareStart string
	MiddlewareEnd   string
	Authentication  string
	Authorization   string
	RateLimiting    string

	// Health check stages
	HealthCheck        string
	HealthCheckPassed  string
	HealthCheckFailed  string
	HealthCheckWarning string

	// Request processing stages
	RequestReceived  string
	RequestValidated string
	RequestProcessed string
	RequestCompleted string
	RequestFailed    string

	// Response processing stages
	ResponseGenerated string
	ResponseSent      string
	ResponseFailed    string

	// Image/File processing stages
	ImageProcessing     string
	FileProcessing      string
	ContentProcessing   string
	ProcessingCompleted string
	ProcessingFailed    string

	// Tool handling stages
	ToolCallsProcessing string
	ToolValidation      string
	ToolExecution       string

	// Stream processing stages
	StreamStart     string
	StreamChunk     string
	StreamCompleted string
	StreamFailed    string
	ChunkProcessing string

	// Correlation and tracking
	CorrelationGenerated string
	TrackingSetup        string
}{
	// Request lifecycle
	Request:     "Request",
	Processing:  "Processing",
	Response:    "Response",
	Validation:  "Validation",
	Selection:   "Selection",
	Preparation: "Preparation",

	// Vendor operations
	VendorRequest:  "VendorRequest",
	VendorResponse: "VendorResponse",
	VendorError:    "VendorError",

	// System operations
	Initialization:    "Initialization",
	Configuration:     "Configuration",
	DatabaseOperation: "DatabaseOperation",

	// Results
	Success:  "Success",
	Error:    "Error",
	Retry:    "Retry",
	Fallback: "Fallback",

	// Middleware stages
	MiddlewareStart: "MiddlewareStart",
	MiddlewareEnd:   "MiddlewareEnd",
	Authentication:  "Authentication",
	Authorization:   "Authorization",
	RateLimiting:    "RateLimiting",

	// Health check stages
	HealthCheck:        "HealthCheck",
	HealthCheckPassed:  "HealthCheckPassed",
	HealthCheckFailed:  "HealthCheckFailed",
	HealthCheckWarning: "HealthCheckWarning",

	// Request processing stages
	RequestReceived:  "RequestReceived",
	RequestValidated: "RequestValidated",
	RequestProcessed: "RequestProcessed",
	RequestCompleted: "RequestCompleted",
	RequestFailed:    "RequestFailed",

	// Response processing stages
	ResponseGenerated: "ResponseGenerated",
	ResponseSent:      "ResponseSent",
	ResponseFailed:    "ResponseFailed",

	// Image/File processing stages
	ImageProcessing:     "ImageProcessing",
	FileProcessing:      "FileProcessing",
	ContentProcessing:   "ContentProcessing",
	ProcessingCompleted: "ProcessingCompleted",
	ProcessingFailed:    "ProcessingFailed",

	// Tool handling stages
	ToolCallsProcessing: "ToolCallsProcessing",
	ToolValidation:      "ToolValidation",
	ToolExecution:       "ToolExecution",

	// Stream processing stages
	StreamStart:     "StreamStart",
	StreamChunk:     "StreamChunk",
	StreamCompleted: "StreamCompleted",
	StreamFailed:    "StreamFailed",
	ChunkProcessing: "ChunkProcessing",

	// Correlation and tracking
	CorrelationGenerated: "CorrelationGenerated",
	TrackingSetup:        "TrackingSetup",
}

// ComponentNames defines standardized component names
var ComponentNames = struct {
	// Core components
	APIRouter  string
	APIClient  string
	Middleware string
	Handler    string
	Proxy      string
	App        string

	// Processing components
	ImageProcessor    string
	FileProcessor     string
	AudioProcessor    string
	StreamProcessor   string
	ResponseProcessor string
	ToolHandler       string

	// Infrastructure components
	Logger     string
	Config     string
	Database   string
	Cache      string
	Monitoring string

	// Vendor components
	OpenAIClient     string
	GeminiClient     string
	VendorHTTPClient string

	// Utility components
	Validator     string
	Selector      string
	Sanitizer     string
	ErrorHandler  string
	RetryExecutor string
}{
	// Core components
	APIRouter:  "APIRouter",
	APIClient:  "APIClient",
	Middleware: "Middleware",
	Handler:    "Handler",
	Proxy:      "Proxy",
	App:        "App",

	// Processing components
	ImageProcessor:    "ImageProcessor",
	FileProcessor:     "FileProcessor",
	AudioProcessor:    "AudioProcessor",
	StreamProcessor:   "StreamProcessor",
	ResponseProcessor: "ResponseProcessor",
	ToolHandler:       "ToolHandler",

	// Infrastructure components
	Logger:     "Logger",
	Config:     "Config",
	Database:   "Database",
	Cache:      "Cache",
	Monitoring: "Monitoring",

	// Vendor components
	OpenAIClient:     "OpenAIClient",
	GeminiClient:     "GeminiClient",
	VendorHTTPClient: "VendorHTTPClient",

	// Utility components
	Validator:     "Validator",
	Selector:      "Selector",
	Sanitizer:     "Sanitizer",
	ErrorHandler:  "ErrorHandler",
	RetryExecutor: "RetryExecutor",
}
