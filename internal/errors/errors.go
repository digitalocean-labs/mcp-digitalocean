package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "validation"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeAuthorization ErrorType = "authorization"
	ErrorTypeNotFound      ErrorType = "not_found"
	ErrorTypeRateLimit     ErrorType = "rate_limit"
	ErrorTypeInternal      ErrorType = "internal"
	ErrorTypeNetwork       ErrorType = "network"
	ErrorTypeTimeout       ErrorType = "timeout"
)

// MCPError represents a structured error for the MCP server
type MCPError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"status_code,omitempty"`
	Retryable  bool      `json:"retryable"`
}

// Error implements the error interface
func (e *MCPError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(message, details string) *MCPError {
	return &MCPError{
		Type:       ErrorTypeValidation,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusBadRequest,
		Retryable:  false,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message, details string) *MCPError {
	return &MCPError{
		Type:       ErrorTypeAuthentication,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusUnauthorized,
		Retryable:  false,
	}
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(message, details string) *MCPError {
	return &MCPError{
		Type:       ErrorTypeAuthorization,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusForbidden,
		Retryable:  false,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource, identifier string) *MCPError {
	return &MCPError{
		Type:       ErrorTypeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		Details:    fmt.Sprintf("No %s found with identifier: %s", resource, identifier),
		StatusCode: http.StatusNotFound,
		Retryable:  false,
	}
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string) *MCPError {
	return &MCPError{
		Type:       ErrorTypeRateLimit,
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
		Retryable:  true,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message, details string) *MCPError {
	return &MCPError{
		Type:       ErrorTypeInternal,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusInternalServerError,
		Retryable:  true,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(message, details string) *MCPError {
	return &MCPError{
		Type:       ErrorTypeNetwork,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusServiceUnavailable,
		Retryable:  true,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string) *MCPError {
	return &MCPError{
		Type:       ErrorTypeTimeout,
		Message:    message,
		StatusCode: http.StatusRequestTimeout,
		Retryable:  true,
	}
}

// WrapError wraps a generic error into an MCPError
func WrapError(err error, errorType ErrorType, message string) *MCPError {
	if err == nil {
		return nil
	}
	
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr
	}
	
	return &MCPError{
		Type:      errorType,
		Message:   message,
		Details:   err.Error(),
		Retryable: errorType == ErrorTypeNetwork || errorType == ErrorTypeTimeout || errorType == ErrorTypeInternal,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr.Retryable
	}
	return false
}

// GetErrorType returns the error type if it's an MCPError
func GetErrorType(err error) ErrorType {
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr.Type
	}
	return ErrorTypeInternal
}
