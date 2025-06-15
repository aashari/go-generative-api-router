package utils

// HTTP Header Constants
const (
	// Standard HTTP Headers
	HeaderContentType     = "Content-Type"
	HeaderContentLength   = "Content-Length"
	HeaderContentEncoding = "Content-Encoding"
	HeaderUserAgent       = "User-Agent"
	HeaderAcceptEncoding  = "Accept-Encoding"
	HeaderCacheControl    = "Cache-Control"
	HeaderConnection      = "Connection"
	HeaderServer          = "Server"

	// Request/Response Tracking Headers
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
	HeaderResponseTime  = "X-Response-Time"

	// Client IP Headers (priority order)
	HeaderXForwardedFor  = "X-Forwarded-For"
	HeaderXRealIP        = "X-Real-IP"
	HeaderCFConnectingIP = "CF-Connecting-IP"
	HeaderCloudFlareRay  = "cf-ray"

	// Security Headers
	HeaderXContentTypeOptions = "X-Content-Type-Options"
	HeaderXFrameOptions       = "X-Frame-Options"
	HeaderXXSSProtection      = "X-XSS-Protection"
	HeaderReferrerPolicy      = "Referrer-Policy"
	HeaderXCSRFToken          = "X-CSRF-Token"

	// Service Headers
	HeaderXPoweredBy      = "X-Powered-By"
	HeaderXVendorSource   = "X-Vendor-Source"
	HeaderXAccelBuffering = "X-Accel-Buffering"

	// Transfer Headers
	HeaderTransferEncoding = "Transfer-Encoding"
	HeaderVary             = "Vary"

	// CORS Headers
	HeaderAccessControlAllowOrigin   = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods  = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders  = "Access-Control-Allow-Headers"
	HeaderAccessControlExposeHeaders = "Access-Control-Expose-Headers"

	// Authorization Headers
	HeaderAuthorization = "Authorization"
)

// Content Type Constants
const (
	ContentTypeJSON            = "application/json"
	ContentTypeJSONUTF8        = "application/json; charset=utf-8"
	ContentTypeEventStream     = "text/event-stream"
	ContentTypeEventStreamUTF8 = "text/event-stream; charset=utf-8"
)

// Cache Control Values
const (
	CacheControlNoCache = "no-cache"
	CacheControlNoStore = "no-cache, no-store, must-revalidate"
)

// Security Header Values
const (
	XContentTypeOptionsNoSniff = "nosniff"
	XFrameOptionsDeny          = "DENY"
	XXSSProtectionBlock        = "1; mode=block"
	ReferrerPolicyStrict       = "strict-origin-when-cross-origin"
)

// Service Values
const (
	ServiceName    = "Generative-API-Router/1.0"
	ServicePowered = "Generative-API-Router"
)

// CORS Values
const (
	CORSAllowOriginAll   = "*"
	CORSAllowMethodsAll  = "POST, GET, OPTIONS, PUT, DELETE"
	CORSAllowHeadersStd  = "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"
	CORSExposeHeadersStd = "X-Request-ID, X-Response-Time"
)

// Transfer Encoding Values
const (
	TransferEncodingChunked = "chunked"
)

// User Agent Patterns
const (
	UserAgentPrefix = "BrainyBuddy-API"
)

// Connection Values
const (
	ConnectionKeepAlive = "keep-alive"
)

// Accept Encoding Values
const (
	AcceptEncodingGzip = "gzip"
)

// System prompt for describing images
const ImageDescriptionPrompt = "Describe the image as detailed as possible. If the image contains text, reproduce it in your response."

// Default model name for image description
const DefaultImageModel = "auto-image-model"

// Header Values for Buffering
const (
	XAccelBufferingNo = "no"
)

// Vary Header Values
const (
	VaryAcceptEncoding = "Accept-Encoding"
)
