package site

import (
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Config holds site-identity configuration.
type Config struct {
	BaseURL         string `validate:"required,url" help:"set SITE_BASE_URL environment variable"`
	OriginAllowlist []string
	// AllowedHosts lists hostnames the server accepts. Built by BuildAllowedHosts
	// from the BaseURL hostname and additional entries.
	AllowedHosts []string `help:"set SITE_BASE_URL or SITE_ALLOWED_HOSTS"`
}

// ValidationRules returns a registrar that enforces site config constraints.
// AllowedHosts must be non-empty — an empty list silently disables host-header
// validation.
func ValidationRules() func(*validator.Validate) {
	return func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			cfg := sl.Current().Interface().(Config)
			if len(cfg.AllowedHosts) == 0 {
				sl.ReportError(cfg.AllowedHosts, "AllowedHosts", "AllowedHosts", "required", "")
			}
		}, Config{})
	}
}

// InitialSiteConfig returns an empty site config. BaseURL must come from env.
func InitialSiteConfig() Config { return Config{} }

// BuildAllowedHosts returns the list of hostnames the server should accept.
// It extracts the hostname from baseURL (if valid) and appends any
// comma-separated entries from extra. The config layer passes raw env values;
// all assembly logic lives here.
func BuildAllowedHosts(baseURL, extra string) []string {
	var hosts []string
	if baseURL != "" {
		u, err := url.Parse(baseURL)
		if err == nil && u.Host != "" {
			hosts = append(hosts, u.Hostname())
		}
	}
	for _, h := range strings.Split(extra, ",") {
		if h = strings.TrimSpace(h); h != "" {
			hosts = append(hosts, h)
		}
	}
	return hosts
}
