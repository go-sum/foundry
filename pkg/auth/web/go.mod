module github.com/go-sum/foundry/pkg/auth/web

go 1.26.0

require (
	github.com/go-sum/foundry/pkg/auth v0.0.0-00010101000000-000000000000
	github.com/go-sum/foundry/pkg/componentry v0.0.0-00010101000000-000000000000
	github.com/go-sum/foundry/pkg/web v0.0.0-00010101000000-000000000000
	github.com/go-sum/foundry/pkg/web/authn v0.0.0-00010101000000-000000000000
	github.com/go-sum/foundry/pkg/web/render v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	maragu.dev/gomponents v1.3.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/go-webauthn/webauthn v0.17.0 // indirect
	github.com/go-webauthn/x v0.2.3 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/gorilla/schema v1.4.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace (
	github.com/go-sum/foundry/pkg/auth v0.0.0-00010101000000-000000000000 => ../
	github.com/go-sum/foundry/pkg/componentry v0.0.0-00010101000000-000000000000 => ../../componentry
	github.com/go-sum/foundry/pkg/web v0.0.0-00010101000000-000000000000 => ../../web
	github.com/go-sum/foundry/pkg/web/authn v0.0.0-00010101000000-000000000000 => ../../web/authn
	github.com/go-sum/foundry/pkg/web/render v0.0.0-00010101000000-000000000000 => ../../web/render
)
