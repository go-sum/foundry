package config

import "net/url"

// ExtractDSNComponents derives PGUSER and PGPASSWORD from DATABASE_URL in env
// for any key that appears in needed but is not already set. Passwords that are
// URL-encoded in the DSN (e.g. "p%40ss") are stored decoded in env.
//
// This keeps DATABASE_URL as the single source of truth: callers provide one
// structured value and individual components are extracted on demand.
func ExtractDSNComponents(env map[string]string, needed map[string]struct{}) {
	dsnStr := env["DATABASE_URL"]
	if dsnStr == "" {
		return
	}

	u, err := url.Parse(dsnStr)
	if err != nil {
		return
	}

	if _, req := needed["PGUSER"]; req {
		if _, exists := env["PGUSER"]; !exists {
			env["PGUSER"] = u.User.Username()
		}
	}

	if _, req := needed["PGPASSWORD"]; req {
		if _, exists := env["PGPASSWORD"]; !exists {
			if p, ok := u.User.Password(); ok {
				env["PGPASSWORD"] = p
			}
		}
	}
}
