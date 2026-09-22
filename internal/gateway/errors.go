package gateway

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorType representa el tipo de error
type ErrorType string

const (
	ErrTypeAuthentication      ErrorType = "authentication_error"
	ErrTypeAuthorization       ErrorType = "authorization_error"
	ErrTypePolicyDenied        ErrorType = "policy_denied"
	ErrTypeRateLimited         ErrorType = "rate_limited"
	ErrTypeQuotaExceeded       ErrorType = "quota_exceeded"
	ErrTypeProviderUnavailable ErrorType = "provider_unavailable"
	ErrTypeProviderTimeout     ErrorType = "provider_timeout"
	ErrTypeProviderError       ErrorType = "provider_error"
	ErrTypeInvalidRequest      ErrorType = "invalid_request"
	ErrTypeInternal            ErrorType = "internal_error"
)

// GatewayError es el error normalizado del gateway
type GatewayError struct {
	Type       ErrorType   `json:"type"`
	Message    string      `json:"message"`
	StatusCode int         `json:"status_code"`
	Details    interface{} `json:"details,omitempty"`
	Cause      error       `json:"-"`
}

func (e *GatewayError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *GatewayError) Unwrap() error {
	return e.Cause
}

// Constructores

func NewAuthenticationError(msg string) *GatewayError {
	return &GatewayError{Type: ErrTypeAuthentication, Message: msg, StatusCode: http.StatusUnauthorized}
}

func NewAuthorizationError(msg string) *GatewayError {
	return &GatewayError{Type: ErrTypeAuthorization, Message: msg, StatusCode: http.StatusForbidden}
}

func NewPolicyDeniedError(msg string) *GatewayError {
	return &GatewayError{Type: ErrTypePolicyDenied, Message: msg, StatusCode: http.StatusForbidden}
}

func NewRateLimitedError(msg string) *GatewayError {
	return &GatewayError{Type: ErrTypeRateLimited, Message: msg, StatusCode: http.StatusTooManyRequests}
}

func NewQuotaExceededError(msg string) *GatewayError {
	return &GatewayError{Type: ErrTypeQuotaExceeded, Message: msg, StatusCode: http.StatusPaymentRequired}
}

func NewProviderUnavailableError(msg string) *GatewayError {
	return &GatewayError{Type: ErrTypeProviderUnavailable, Message: msg, StatusCode: http.StatusServiceUnavailable}
}

func NewProviderTimeoutError(msg string) *GatewayError {
	return &GatewayError{Type: ErrTypeProviderTimeout, Message: msg, StatusCode: http.StatusGatewayTimeout}
}

func NewProviderError(msg string, cause error) *GatewayError {
	return &GatewayError{Type: ErrTypeProviderError, Message: msg, StatusCode: http.StatusBadGateway, Cause: cause}
}

func NewInvalidRequestError(msg string) *GatewayError {
	return &GatewayError{Type: ErrTypeInvalidRequest, Message: msg, StatusCode: http.StatusBadRequest}
}

func NewInternalError(msg string, cause error) *GatewayError {
	return &GatewayError{Type: ErrTypeInternal, Message: msg, StatusCode: http.StatusInternalServerError, Cause: cause}
}

// AsGatewayError convierte cualquier error a GatewayError
func AsGatewayError(err error) *GatewayError {
	if err == nil {
		return nil
	}
	var gwErr *GatewayError
	if errors.As(err, &gwErr) {
		return gwErr
	}
	return NewInternalError("unexpected error", err)
}
