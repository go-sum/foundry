package auth

import (
	"cmp"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

type MethodName string

const (
	MethodEmailTOTP MethodName = "email_totp"
	MethodPasskey   MethodName = "passkey"
)

type Config struct {
	Preferred MethodName      `validate:"omitempty,oneof=email_totp passkey"`
	EmailTOTP EmailTOTPConfig
	Passkey   PasskeyConfig
	Token     TokenConfig
}

type EmailTOTPConfig struct {
	Enabled       bool
	Issuer        string
	PeriodSeconds int
}

type PasskeyConfig struct {
	Enabled               bool
	RPDisplayName         string        `validate:"required_if=Enabled true"`
	RPID                  string        `validate:"required_if=Enabled true"`
	RPOrigins             []string      `validate:"required_if=Enabled true"`
	ResidentKey           string        `validate:"omitempty,oneof=required preferred discouraged"`
	UserVerification      string        `validate:"omitempty,oneof=required preferred discouraged"`
	RegistrationTimeout   time.Duration `validate:"min=0"`
	AuthenticationTimeout time.Duration `validate:"min=0"`
}

type TokenConfig struct {
	Secrets [][]byte      `validate:"required,min=1,dive,min=32"`
	TTL     time.Duration
}

// InitialAuthConfig returns the package's sane defaults for Config.
// Use this as the starting point and override individual fields as needed.
func InitialAuthConfig() Config {
	return Config{
		Preferred: MethodEmailTOTP,
		EmailTOTP: EmailTOTPConfig{
			PeriodSeconds: 300,
		},
		Passkey: PasskeyConfig{
			ResidentKey:           "required",
			UserVerification:      "required",
			RegistrationTimeout:   5 * time.Minute,
			AuthenticationTimeout: 2 * time.Minute,
		},
		Token: TokenConfig{
			TTL: 300 * time.Second,
		},
	}
}

// ApplyDefaults returns cfg with zero-valued fields filled from InitialAuthConfig.
// Deprecated: use InitialAuthConfig and override fields directly.
func ApplyDefaults(cfg Config) Config {
	defaults := InitialAuthConfig()
	cfg.Preferred = cmp.Or(cfg.Preferred, defaults.Preferred)
	cfg.EmailTOTP.PeriodSeconds = cmp.Or(cfg.EmailTOTP.PeriodSeconds, defaults.EmailTOTP.PeriodSeconds)
	cfg.Passkey.ResidentKey = cmp.Or(cfg.Passkey.ResidentKey, defaults.Passkey.ResidentKey)
	cfg.Passkey.UserVerification = cmp.Or(cfg.Passkey.UserVerification, defaults.Passkey.UserVerification)
	if cfg.Passkey.RegistrationTimeout == 0 {
		cfg.Passkey.RegistrationTimeout = defaults.Passkey.RegistrationTimeout
	}
	if cfg.Passkey.AuthenticationTimeout == 0 {
		cfg.Passkey.AuthenticationTimeout = defaults.Passkey.AuthenticationTimeout
	}
	if cfg.Token.TTL == 0 {
		cfg.Token.TTL = time.Duration(cfg.EmailTOTP.PeriodSeconds) * time.Second
	}
	return cfg
}

// PreferredMethod resolves the preferred (top-of-page) auth method for rendering.
// Falls back to MethodEmailTOTP when unset.
func (c Config) PreferredMethod() MethodName {
	return cmp.Or(c.Preferred, MethodEmailTOTP)
}

// RegisterValidationRules registers cross-field validation rules for Config on v.
func (c Config) RegisterValidationRules(v *validator.Validate) {
	v.RegisterStructValidation(authConfigRules, Config{})
}

func authConfigRules(sl validator.StructLevel) {
	cfg := sl.Current().Interface().(Config)

	// TOTP is the baseline account recovery method and must always be enabled.
	// Passkey is additive; it cannot be the only enabled method.
	if !cfg.EmailTOTP.Enabled {
		sl.ReportError(cfg.EmailTOTP, "EmailTOTP", "EmailTOTP", "totp_must_be_enabled", "")
		return
	}

	if cfg.PreferredMethod() == MethodPasskey && !cfg.Passkey.Enabled {
		sl.ReportError(cfg.Preferred, "Preferred", "Preferred", "preferred_method_disabled", "")
	}
}

// Validate checks cross-field constraints for PasskeyConfig.
// Each RPOrigin must be a valid HTTPS URL (except localhost/127.0.0.1 which may use HTTP)
// whose hostname matches RPID or is a subdomain of RPID.
func (c PasskeyConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	for _, origin := range c.RPOrigins {
		u, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("passkey config: invalid RPOrigin %q: %w", origin, err)
		}
		host := u.Hostname()
		isLocalhost := host == "localhost" || host == "127.0.0.1"
		if u.Scheme == "http" && !isLocalhost {
			return fmt.Errorf("passkey config: RPOrigin %q uses http for non-localhost host", origin)
		}
		if host != c.RPID && !strings.HasSuffix(host, "."+c.RPID) {
			return fmt.Errorf("passkey config: RPOrigin %q hostname %q does not match RPID %q", origin, host, c.RPID)
		}
	}
	return nil
}
