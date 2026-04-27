package notification_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sum/foundry/pkg/notification"
)

func TestSentinelErrors_ErrorsIs(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
	}{
		{"ErrProviderUnknown", notification.ErrProviderUnknown},
		{"ErrChannelUnavailable", notification.ErrChannelUnavailable},
		{"ErrInvalidConfig", notification.ErrInvalidConfig},
		{"ErrDeliveryFailed", notification.ErrDeliveryFailed},
		{"ErrTransient", notification.ErrTransient},
		{"ErrQueueFull", notification.ErrQueueFull},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("wrapping: %w", tt.sentinel)
			if !errors.Is(wrapped, tt.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false, want true", wrapped, tt.sentinel)
			}
		})
	}
}

func TestSentinelErrors_NotEqual(t *testing.T) {
	sentinels := []error{
		notification.ErrProviderUnknown,
		notification.ErrChannelUnavailable,
		notification.ErrInvalidConfig,
		notification.ErrDeliveryFailed,
		notification.ErrTransient,
		notification.ErrQueueFull,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("errors.Is(%v, %v) = true, want false (sentinels must be distinct)", a, b)
			}
		}
	}
}
