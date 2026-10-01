package errorx

import "errors"

var (
	ErrNotFound         = errors.New("notification not found")
	ErrInvalidID        = errors.New("invalid notification id")
	ErrInvalidRecipient = errors.New("invalid recipient")
	ErrNotCancellable   = errors.New("notification can no longer be cancelled")
)
