package domain

import "errors"

var (
	ErrInvalidActivityID        = errors.New("invalid activity id")
	ErrDetailedActivityNotFound = errors.New("detailed activity not found")
)
