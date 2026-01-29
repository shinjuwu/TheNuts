package domain

import "errors"

var (
	ErrNotYourTurn       = errors.New("not your turn")
	ErrCannotCheck       = errors.New("cannot check: there is an outstanding bet")
	ErrBetTooLow         = errors.New("bet amount is below minimum")
	ErrInsufficientChips = errors.New("insufficient chips")
	ErrAlreadyAllIn      = errors.New("already all-in or no chips")
)
