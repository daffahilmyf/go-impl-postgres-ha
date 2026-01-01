package repository

import "errors"

var ErrIdempotencyKeyConflict = errors.New("idempotency key conflicts with request")
var ErrInvalidCursor = errors.New("invalid cursor")
