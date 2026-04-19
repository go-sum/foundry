module github.com/go-sum/foundry

go 1.26.0

require (
	github.com/go-sum/assets v0.0.0
	github.com/go-sum/componentry v0.0.0
	github.com/go-sum/config v0.0.0
	github.com/go-sum/web v0.0.0
	github.com/go-sum/web/render v0.0.0
	go.opentelemetry.io/otel/trace v1.43.0
	maragu.dev/gomponents v1.3.0
)

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace (
	github.com/go-sum/assets => ../pkg/assets
	github.com/go-sum/componentry => ../pkg/componentry
	github.com/go-sum/config => ../pkg/config
	github.com/go-sum/web => ../pkg/web
	github.com/go-sum/web/render => ../pkg/web/render
)
