package config

func devOverlay(cfg *Config) {
	cfg.Security.CSRF.CookieSecure = false
}
