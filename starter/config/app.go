package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/notification/email"
)

var validEmailProviders = map[email.Provider]struct{}{
	email.ProviderResend:       {},
	email.ProviderMailChannels: {},
	email.ProviderLog:          {},
}

// AppConfig holds application-domain configuration.
type AppConfig struct {
	CaptureErrorStacks bool
	Contact            ContactConfig
	Email              email.Config
}

// ContactConfig holds configuration for the contact feature.
type ContactConfig struct {
	SendTo   string
	SendFrom string
}

func productionApp() (AppConfig, error) {
	captureStacks, err := cfgpkg.ExpandEnvBool("APP_CAPTURE_ERROR_STACKS", true)
	if err != nil {
		return AppConfig{}, fmt.Errorf("config: APP_CAPTURE_ERROR_STACKS: %w", err)
	}
	return AppConfig{
		CaptureErrorStacks: captureStacks,
		Contact: ContactConfig{
			SendTo:   cfgpkg.ExpandEnv("EMAIL_SEND_TO", "send@example.com"),
			SendFrom: cfgpkg.ExpandEnv("EMAIL_SEND_FROM", "noreply@example.com"),
		},
		Email: email.Config{
			Provider: email.Provider(cfgpkg.ExpandEnv("EMAIL_PROVIDER", "")),
			APIKey:   cfgpkg.ExpandSecret("EMAIL_API_KEY"),
			From:     cfgpkg.ExpandEnv("EMAIL_SEND_FROM", "noreply@example.com"),
		},
	}, nil
}

// emailProviderRules returns a validator registrar that requires an explicit
// email provider selection from the supported provider set.
func emailProviderRules(provider email.Provider) func(*validator.Validate) {
	return func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			if provider == "" {
				sl.ReportError(provider, "EmailProvider", "EmailProvider", "required", "")
				return
			}
			if _, ok := validEmailProviders[provider]; !ok {
				sl.ReportError(provider, "EmailProvider", "EmailProvider", "oneof", "resend mailchannels log")
			}
		}, AppConfig{})
	}
}
