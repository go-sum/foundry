package config

import (
	"github.com/go-playground/validator/v10"
	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/notification/email"
)

// AppConfig holds application-domain configuration.
type AppConfig struct {
	Contact ContactConfig
	Email   email.Config
}

// ContactConfig holds configuration for the contact feature.
type ContactConfig struct {
	SendTo   string
	SendFrom string
}

func productionApp() AppConfig {
	return AppConfig{
		Contact: ContactConfig{
			SendTo:   cfgpkg.ExpandEnv("EMAIL_SEND_TO", "send@example.com"),
			SendFrom: cfgpkg.ExpandEnv("EMAIL_SEND_FROM", "noreply@example.com"),
		},
		Email: email.Config{
			Provider: email.Provider(cfgpkg.ExpandEnv("EMAIL_PROVIDER", "")),
			APIKey:   cfgpkg.ExpandSecret("EMAIL_API_KEY"),
			From:     cfgpkg.ExpandEnv("EMAIL_SEND_FROM", "noreply@example.com"),
		},
	}
}

// emailProviderRules returns a validator registrar that prevents the "log"
// email provider from being used in production.
func emailProviderRules(provider email.Provider, env string) func(*validator.Validate) {
	return func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			if env == string(Production) && (provider == email.ProviderLog || provider == "") {
				sl.ReportError(provider, "EmailProvider", "EmailProvider", "email_provider_production_required", "")
			}
		}, AppConfig{})
	}
}
