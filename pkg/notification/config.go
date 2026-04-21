package notification

import "time"

// Config configures the notification system. Use config.Validate[Config] at
// the composition root to validate before constructing a Dispatcher.
type Config struct {
	DefaultChannels []Channel                `validate:"dive,oneof=email webhook log" help:"channels used when not specified per-notification"`
	Timeout         time.Duration            `help:"per-channel send timeout (default 10s)"`
	Providers       map[string]ProviderConfig `validate:"dive"                        help:"named provider configurations"`
}

// ProviderConfig holds per-provider settings.
type ProviderConfig struct {
	Channel  Channel           `validate:"required,oneof=email webhook log" help:"delivery channel this provider serves"`
	Settings map[string]string `help:"provider-specific key-value settings"`
}
