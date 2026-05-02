package config

import (
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
			Provider: "log",
			APIKey:   cfgpkg.ExpandSecret("EMAIL_API_KEY"),
			From:     cfgpkg.ExpandEnv("EMAIL_SEND_FROM", "noreply@example.com"),
		},
	}
}
