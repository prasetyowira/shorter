package constant

// Request context keys
const (
	RequestIDKey = "request_id"
)

// HTTP header names
const (
	HeaderRequestID = "X-Request-ID"
)

// Function/Context names
const (
	// Domain context names
	CtxDomain         = "domain"
	CtxCreateShortURL = "CreateShortURL"
	CtxGetLongURL     = "GetLongURL"

	// Infrastructure context names
	CtxDB              = "db"
	CtxStore           = "Store"
	CtxFindByShortCode = "FindByShortCode"
	CtxIncrementVisits = "IncrementVisits"
	CtxClose           = "Close"
	CtxAPI             = "api"

	// General context names
	CtxRouter            = "Router"
	CtxMain              = "Main"
	CtxRedirectToLongURL = "RedirectToLongURL"
	CtxGetURLStats       = "GetURLStats"
)

// Data field keys
const (
	// Service data fields
	DataService     = "service"
	DataLongURL     = "long_url"
	DataCustomShort = "custom_short"
	DataShortCode   = "short_code"
	DataCustom      = "custom"
	DataVisits      = "visits"

	// Database data fields
	DataPath         = "path"
	DataElapsed      = "elapsed"
	DataRows         = "rows"
	DataSQL          = "sql"
	DataData         = "data"
	DataRowsAffected = "rows_affected"

	// API data fields
	DataMethod      = "method"
	DataIP          = "ip"
	DataAgent       = "agent"
	DataStatus      = "status"
	DataLatency     = "latency"
	DataSize        = "size"
	DataRemoteAddr  = "remote_addr"
	DataUserAgent   = "user_agent"
	DataPort        = "port"
	DataDBPath      = "db_path"
	DataEnvironment = "environment"
)

// Error message constants
const (
	ErrEmptyLongURL      = "Long URL cannot be empty"
	ErrEmptyShortCode    = "Short code cannot be empty"
	ErrShortCodeExists   = "short code already exists"
	ErrShortCodeNotFound = "short code not found"
)

// Error codes
const (
	ErrCodeAPIDecodeRequest  = "API001"
	ErrCodeAPIServiceError   = "API002"
	ErrCodeAppDBInit         = "APP001"
	ErrCodeAppServerStart    = "APP002"
	ErrCodeAppServerShutdown = "APP003"
)

// Error types
const (
	ErrTypeDomain = "domain"
	ErrTypeAPI    = "api"
	ErrTypeApp    = "application"
)

// API routes
const (
	RouteCreateShortURL    = "/api/urls"
	RouteShortCodeRedirect = "/{shortCode}"
	RouteURLStats          = "/api/urls/{shortCode}/stats"
	RouteHealthcheck       = "/health"
)

// Log keys
const (
	LogTimeKey         = "time"
	LogLevelKey        = "level"
	LogNameKey         = "logger"
	LogCallerKey       = "caller"
	LogMessageKey      = "msg"
	LogStacktraceKey   = "stacktrace"
	LogRequestIDKey    = "request_id"
	LogFunctionKey     = "function"
	LogErrorCodeKey    = "error_code"
	LogErrorTypeKey    = "error_type"
	LogErrorMessageKey = "error_message"
	LogEncodingJSON    = "json"
	LogEncodingConsole = "console"
	LogOutputStdout    = "stdout"
	LogOutputStderr    = "stderr"
)

// Environment constants
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Message constants for application
const (
	MsgApplicationStarting       = "Application starting"
	MsgFailedToInitDB            = "Failed to initialize database"
	MsgServerStarting            = "Server starting"
	MsgServerFailedToStart       = "Server failed to start"
	MsgServerShuttingDown        = "Server shutting down"
	MsgServerShutdownError       = "Error during server shutdown"
	MsgServerStopped             = "Server stopped"
	MsgRequestReceived           = "Request received"
	MsgHandlingCreateRequest     = "Handling create short URL request"
	MsgProcessingRedirectRequest = "Processing URL redirection request"
	MsgSettingUpRoutes           = "Setting up API routes"
	MsgHealthcheckRequest        = "Handling healthcheck request"
	MsgHealthy                   = "Healthy"
	MsgRequestCompleted          = "Request completed"
)

// Cache Namespace
const (
	ShortURLNamespace = "SHORT"
)
