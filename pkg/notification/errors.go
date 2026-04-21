package notification

import "errors"

var (
	ErrProviderUnknown    = errors.New("notification: unknown provider")
	ErrChannelUnavailable = errors.New("notification: channel unavailable")
	ErrInvalidConfig      = errors.New("notification: invalid configuration")
	ErrDeliveryFailed     = errors.New("notification: delivery failed")
	ErrTransient          = errors.New("notification: transient failure")
	ErrQueueFull          = errors.New("notification: send queue full")
)
