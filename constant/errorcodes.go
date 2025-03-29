package constant

// Domain service error codes
const (
	// Shortener service - Validation errors (1xx)
	ErrCodeEmptyLongURL   = "SVC001"
	ErrCodeEmptyShortCode = "SVC003"
	
	// Shortener service - Storage errors (2xx)
	ErrCodeStorageFailure = "SVC002"
	
	// Shortener service - Retrieval errors (3xx)
	ErrCodeShortCodeNotFound = "SVC004"
	
	// Shortener service - Stats errors (4xx)
	ErrCodeIncrementVisits = "SVC005"
)

// Database error codes
const (
	// General DB errors (5xx)
	ErrCodeDBGeneral = "DB500"
	
	// Connection errors (0xx)
	ErrCodeDBOpen    = "DB001"
	ErrCodeDBMigrate = "DB002"
	
	// Store operation errors (1xx)
	ErrCodeDBCheckExists = "DB101"
	ErrCodeDBInsert      = "DB102"
	
	// FindByShortCode operation errors (2xx)
	ErrCodeDBLookup     = "DB201"
	ErrCodeDBScanRows   = "DB202"
	ErrCodeDBRowIterate = "DB203"
	
	// IncrementVisits operation errors (3xx)
	ErrCodeDBIncrement = "DB301"
	
	// Close operation errors (4xx)
	ErrCodeDBClose = "DB401"
)

// Error types for categorization
const (
	// Domain error types
	ErrTypeValidation = "validation"
	ErrTypeStorage    = "storage"
	ErrTypeRetrieval  = "retrieval"
	ErrTypeStats      = "stats"
	
	// Infrastructure error types
	ErrTypeDB = "db"
) 