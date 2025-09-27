package batchsql

// ConflictStrategy defines how to handle conflicts during batch operations
type ConflictStrategy int

const (
	// ConflictIgnore ignores conflicts and continues
	ConflictIgnore ConflictStrategy = iota
	// ConflictReplace replaces existing records
	ConflictReplace
	// ConflictUpdate updates existing records
	ConflictUpdate
	// ConflictError returns an error on conflict
	ConflictError
)

// String returns the string representation of ConflictStrategy
func (cs ConflictStrategy) String() string {
	switch cs {
	case ConflictIgnore:
		return "IGNORE"
	case ConflictReplace:
		return "REPLACE"
	case ConflictUpdate:
		return "UPDATE"
	case ConflictError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// DatabaseType represents the type of database
type DatabaseType int

const (
	// DatabaseTypeMySQL represents MySQL database
	DatabaseTypeMySQL DatabaseType = iota
	// DatabaseTypePostgreSQL represents PostgreSQL database
	DatabaseTypePostgreSQL
	// DatabaseTypeSQLite represents SQLite database
	DatabaseTypeSQLite
	// DatabaseTypeRedis represents Redis database
	DatabaseTypeRedis
	// DatabaseTypeMongoDB represents MongoDB database
	DatabaseTypeMongoDB
)

// String returns the string representation of DatabaseType
func (dt DatabaseType) String() string {
	switch dt {
	case DatabaseTypeMySQL:
		return "MySQL"
	case DatabaseTypePostgreSQL:
		return "PostgreSQL"
	case DatabaseTypeSQLite:
		return "SQLite"
	case DatabaseTypeRedis:
		return "Redis"
	case DatabaseTypeMongoDB:
		return "MongoDB"
	default:
		return "Unknown"
	}
}

// ExecutionResult represents the result of a batch execution
type ExecutionResult struct {
	Success      bool                   `json:"success"`
	AffectedRows int64                  `json:"affected_rows"`
	Duration     int64                  `json:"duration_ms"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	return ve.Message
}

// BatchError represents an error that occurred during batch processing
type BatchError struct {
	Message string             `json:"message"`
	Errors  []*ValidationError `json:"errors,omitempty"`
	Code    string             `json:"code,omitempty"`
}

// Error implements the error interface
func (be *BatchError) Error() string {
	return be.Message
}

// AddError adds a validation error to the batch error
func (be *BatchError) AddError(field, message string, value interface{}) {
	be.Errors = append(be.Errors, &ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are any errors
func (be *BatchError) HasErrors() bool {
	return len(be.Errors) > 0
}
