package errors

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrorResponse represents the standardized error response format
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ToErrorResponse converts an AppError to ErrorResponse
func ToErrorResponse(err error) *ErrorResponse {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		// Unknown error, wrap it
		appErr = Wrap(err, CodeInternalError, CategorySystem, "An unexpected error occurred")
	}

	return &ErrorResponse{
		Error:   appErr.Category,
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	}
}

// HTTPStatusCode returns the appropriate HTTP status code for an error
func HTTPStatusCode(err error) int {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		return http.StatusInternalServerError
	}

	switch appErr.Code {
	// Validation Errors -> 400 Bad Request
	case CodeInvalidRequest, CodeInvalidPhoneNumber, CodeContentTooLong,
		CodeInvalidTenantID, CodeInvalidMessageID, CodeInvalidStatus:
		return http.StatusBadRequest

	// Business Logic Errors
	case CodeInsufficientBalance:
		return http.StatusPaymentRequired // 402
	case CodeTenantNotFound, CodeMessageNotFound, CodeChannelNotFound:
		return http.StatusNotFound // 404
	case CodeTenantSuspended:
		return http.StatusForbidden // 403
	case CodeNoChannelAvailable, CodeChannelOffline, CodeChannelBusy:
		return http.StatusServiceUnavailable // 503
	case CodeDuplicateReport:
		return http.StatusConflict // 409
	case CodeRateLimitExceeded:
		return http.StatusTooManyRequests // 429

	// Database Errors -> 500 Internal Server Error
	case CodeDatabaseConnection, CodeDatabaseQuery, CodeDatabaseTransaction,
		CodeConcurrentUpdate, CodeDeadlock:
		return http.StatusInternalServerError

	// Network Errors -> 503 Service Unavailable
	case CodeConnectionClosed, CodeConnectionTimeout, CodeSendTimeout, CodeBufferFull:
		return http.StatusServiceUnavailable

	// System Errors -> 500 Internal Server Error
	case CodeInternalError, CodeServiceUnavailable, CodeCircuitBreakerOpen:
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}
}

// WriteJSONError writes an error response as JSON
func WriteJSONError(w http.ResponseWriter, err error) {
	statusCode := HTTPStatusCode(err)
	response := ToErrorResponse(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
