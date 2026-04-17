package config

import (
	"errors"
	"fmt"

	cfgpkg "github.com/go-sum/config"
	"github.com/go-sum/web/secure"
)

type SecureConfig struct {
	CSRF secure.CSRFConfig
}

func defaultSecure() (SecureConfig, error) {
	keyHex := cfgpkg.ExpandSecret("SECURITY_CSRF_KEY")
	if keyHex == "" {
		return SecureConfig{}, fmt.Errorf("%w: set SECURITY_CSRF_KEY environment variable", ErrCSRFKeyMissing)
	}

	csrf, err := secure.NewCSRFConfigFromHex(
		keyHex,
		cfgpkg.ExpandSecret("SECURITY_CSRF_KEY_PREVIOUS"),
	)
	if err != nil {
		if errors.Is(err, secure.ErrCSRFPreviousKeys) {
			return SecureConfig{}, fmt.Errorf("%w: %w", ErrCSRFPrevKeysInvalid, err)
		}
		return SecureConfig{}, fmt.Errorf("%w: %w", ErrCSRFKeyInvalid, err)
	}
	csrf.CookieSecure = true
	return SecureConfig{CSRF: csrf}, nil
}

