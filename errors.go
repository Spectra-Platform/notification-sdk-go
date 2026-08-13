package notification

import (
	"errors"
	"fmt"
	"time"
)

const (
	CodeMalformedResponse  = "MALFORMED_RESPONSE"
	CodeNetworkUnavailable = "NETWORK_UNAVAILABLE"
	CodeValidationError    = "VALIDATION_ERROR"

	CodeValidationFailed             = "VALIDATION_FAILED"
	CodeDeliveryRateLimited          = "DELIVERY_RATE_LIMITED"
	CodeProjectTokenUnauthorized     = "PROJECT_TOKEN_UNAUTHORIZED"
	CodeAuthIntrospectionUnavailable = "AUTH_INTROSPECTION_UNAVAILABLE"
	CodeIdempotencyConflict          = "IDEMPOTENCY_CONFLICT"

	CodeEmailResourceNotFound        = "EMAIL_RESOURCE_NOT_FOUND"
	CodeEmailVerificationInvalid     = "EMAIL_VERIFICATION_INVALID"
	CodeEmailVerificationExpired     = "EMAIL_VERIFICATION_EXPIRED"
	CodeEmailVerificationConsumed    = "EMAIL_VERIFICATION_CONSUMED"
	CodeEmailVerificationLocked      = "EMAIL_VERIFICATION_LOCKED"
	CodeEmailTemplatePublishConflict = "EMAIL_TEMPLATE_PUBLISH_CONFLICT"
)

type Error struct {
	Code       string
	Message    string
	HTTPStatus int
	RequestID  string
	RetryAfter time.Duration
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.HTTPStatus > 0 {
		return fmt.Sprintf("%s: %s (http %d)", e.Code, e.Message, e.HTTPStatus)
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsCode(err error, code string) bool {
	var sdkError *Error
	if !errors.As(err, &sdkError) {
		return false
	}
	return sdkError.Code == code
}

func newValidationError(message string) *Error {
	return &Error{
		Code:    CodeValidationError,
		Message: message,
	}
}

func newNetworkUnavailableError(cause error) *Error {
	return &Error{
		Code:    CodeNetworkUnavailable,
		Message: "could not reach Spectra Delivery",
		Err:     cause,
	}
}

func newMalformedResponseError(httpStatus int, requestID string, cause error) *Error {
	message := "delivery API returned a malformed response"
	if cause != nil {
		message = fmt.Sprintf("%s: %v", message, cause)
	}

	return &Error{
		Code:       CodeMalformedResponse,
		Message:    message,
		HTTPStatus: httpStatus,
		RequestID:  requestID,
		Err:        cause,
	}
}
