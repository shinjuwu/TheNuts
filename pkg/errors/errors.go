package errors

import "errors"

var (
	ErrInvalidAction  = errors.New("INVALID_ACTION")
	ErrTableNotFound  = errors.New("TABLE_NOT_FOUND")
	ErrPlayerNotFound = errors.New("PLAYER_NOT_FOUND")
	ErrUnauthorized   = errors.New("UNAUTHORIZED")
	ErrInternal       = errors.New("INTERNAL_ERROR")
)
