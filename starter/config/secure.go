package config

import (
	"errors"
	"fmt"

	cfgpkg "github.com/go-sum/config"
	"github.com/go-sum/web/secure"
)

func defaultCSRF() (secure.CSRFConfig, error) {
	keyHex := cfgpkg.ExpandSecret("SECURITY_CSRF_KEY")
	if keyHex == "" {
		return secure.CSRFConfig{}, fmt.Errorf("%w: set SECURITY_CSRF_KEY environment variable", ErrCSRFKeyMissing)
	}
	csrf, err := secure.NewCSRFConfigFromHex(keyHex, cfgpkg.ExpandSecret("SECURITY_CSRF_KEY_PREVIOUS"))
	if err != nil {
		if errors.Is(err, secure.ErrCSRFPreviousKeys) {
			return secure.CSRFConfig{}, fmt.Errorf("%w: %w", ErrCSRFPrevKeysInvalid, err)
		}
		return secure.CSRFConfig{}, fmt.Errorf("%w: %w", ErrCSRFKeyInvalid, err)
	}
	csrf.CookieSecure = true
	return csrf, nil
}
