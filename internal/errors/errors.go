package errors

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrQueueEmpty   = errors.New("queue empty")
	ErrValidation   = errors.New("validation failed")
	ErrConflict     = errors.New("conflict")
	ErrInvalidState = errors.New("invalid state")
)
