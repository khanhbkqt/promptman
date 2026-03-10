package envelope

import "net/http"

// Error code constants used across all modules.
const (
	// Collection errors (M1)
	CodeCollectionNotFound = "COLLECTION_NOT_FOUND"
	CodeRequestNotFound    = "REQUEST_NOT_FOUND"

	// Environment errors (M2)
	CodeEnvNotFound         = "ENV_NOT_FOUND"
	CodeEnvNotSet           = "ENV_NOT_SET"
	CodeSecretResolveFailed = "SECRET_RESOLVE_FAILED"

	// Validation errors (multiple modules)
	CodeInvalidYAML    = "INVALID_YAML"
	CodeInvalidRequest = "INVALID_REQUEST"
	CodeInvalidInput   = "INVALID_INPUT"

	// Request execution errors (M3)
	CodeRequestTimeout = "REQUEST_TIMEOUT"
	CodeRequestFailed  = "REQUEST_FAILED"

	// Test execution errors (M4)
	CodeTestExecutionError = "TEST_EXECUTION_ERROR"

	// Import/export errors (M11)
	CodeImportFailed = "IMPORT_FAILED"
	CodeExportFailed = "EXPORT_FAILED"

	// Daemon errors
	CodeDaemonBusy           = "DAEMON_BUSY"
	CodeDaemonAlreadyRunning = "DAEMON_ALREADY_RUNNING"
	CodeDaemonNotRunning     = "DAEMON_NOT_RUNNING"
	CodeLockFileCorrupt      = "LOCK_FILE_CORRUPT"
	CodePortUnavailable      = "PORT_UNAVAILABLE"

	// Generic errors
	CodeInternalError = "INTERNAL_ERROR"
	CodeNotFound      = "NOT_FOUND"
	CodeUnauthorized  = "UNAUTHORIZED"
)

// HTTPStatusForCode returns the default HTTP status code for an error code.
// Returns 500 for unknown codes.
func HTTPStatusForCode(code string) int {
	switch code {
	case CodeCollectionNotFound, CodeRequestNotFound, CodeEnvNotFound, CodeNotFound:
		return http.StatusNotFound
	case CodeEnvNotSet, CodeInvalidYAML, CodeInvalidRequest, CodeInvalidInput,
		CodeImportFailed, CodeExportFailed:
		return http.StatusBadRequest
	case CodeRequestTimeout:
		return http.StatusRequestTimeout
	case CodeRequestFailed:
		return http.StatusBadGateway
	case CodeTestExecutionError, CodeInternalError, CodeSecretResolveFailed:
		return http.StatusInternalServerError
	case CodeDaemonBusy:
		return http.StatusTooManyRequests
	case CodeDaemonAlreadyRunning:
		return http.StatusConflict
	case CodeDaemonNotRunning, CodeLockFileCorrupt:
		return http.StatusBadRequest
	case CodePortUnavailable:
		return http.StatusServiceUnavailable
	case CodeUnauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
