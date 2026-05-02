package email

import "time"

// Provider identifies an email delivery service.
type Provider string

const (
	ProviderResend       Provider = "resend"
	ProviderMailChannels Provider = "mailchannels"
	ProviderLog          Provider = "log"
)

const defaultTimeout = 10 * time.Second

// Config configures an email Sender. Provider selects the implementation;
// APIKey and BaseURL are provider-specific. From is the default sender address
// used when Message.From is empty.
type Config struct {
	Provider Provider
	APIKey   string
	BaseURL  string        // optional; each provider has its own default URL
	From     string        // default sender address
	Timeout  time.Duration // HTTP timeout; defaults to 10s
}

// InitialEmailConfig returns a Config with safe defaults suitable for
// development and testing. The "log" provider requires no API keys.
func InitialEmailConfig() Config {
	return Config{
		Provider: ProviderLog,
		Timeout:  defaultTimeout,
	}
}
