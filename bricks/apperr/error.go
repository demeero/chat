package apperr

import "errors"

var (
	// ErrInvalidData is returned when the input data is invalid.
	ErrInvalidData = errors.New("invalid data")
	// ErrNotFound is returned when the requested resource is not found.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when the requested resource already exists or some another data conflict occurs.
	ErrConflict = errors.New("conflict")
	// ErrForbidden is returned when the user is not authorized to perform the requested action.
	ErrForbidden = errors.New("forbidden")
	// ErrUnauthorized is returned when the user is not correctly authenticated.
	ErrUnauthorized = errors.New("unauthorized")
)
