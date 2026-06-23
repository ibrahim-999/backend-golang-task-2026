package errs

import (
	"errors"
	"fmt"
)

type Kind string

const (
	KindInternal     Kind = "internal"
	KindValidation   Kind = "validation"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindOutOfStock   Kind = "out_of_stock"
	KindPayment      Kind = "payment_failed"
	KindUnavailable  Kind = "unavailable"
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

func (e *Error) WithCause(err error) *Error {
	e.cause = err
	return e
}

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func Validation(code, message string) *Error    { return New(KindValidation, code, message) }
func NotFound(code, message string) *Error      { return New(KindNotFound, code, message) }
func Conflict(code, message string) *Error      { return New(KindConflict, code, message) }
func Unauthorized(code, message string) *Error  { return New(KindUnauthorized, code, message) }
func Forbidden(code, message string) *Error     { return New(KindForbidden, code, message) }
func OutOfStock(code, message string) *Error    { return New(KindOutOfStock, code, message) }
func PaymentFailed(code, message string) *Error { return New(KindPayment, code, message) }
func Internal(code, message string) *Error      { return New(KindInternal, code, message) }
func Unavailable(code, message string) *Error   { return New(KindUnavailable, code, message) }

func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}

func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
