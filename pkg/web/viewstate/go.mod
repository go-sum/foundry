module github.com/go-sum/foundry/pkg/web/viewstate

go 1.26.0

require (
	github.com/go-sum/foundry/pkg/componentry v0.0.0-00010101000000-000000000000
	github.com/go-sum/foundry/pkg/web v0.0.0-00010101000000-000000000000
	github.com/go-sum/foundry/pkg/web/authn v0.0.0-00010101000000-000000000000
	github.com/go-sum/foundry/pkg/web/render v0.0.0-00010101000000-000000000000
	maragu.dev/gomponents v1.3.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace (
	github.com/go-sum/foundry/pkg/auth v0.0.0-00010101000000-000000000000 => ../../auth
	github.com/go-sum/foundry/pkg/componentry v0.0.0-00010101000000-000000000000 => ../../componentry
	github.com/go-sum/foundry/pkg/web v0.0.0-00010101000000-000000000000 => ../
	github.com/go-sum/foundry/pkg/web/authn v0.0.0-00010101000000-000000000000 => ../authn
	github.com/go-sum/foundry/pkg/web/render v0.0.0-00010101000000-000000000000 => ../render
)
