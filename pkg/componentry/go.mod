module github.com/go-sum/componentry

go 1.26.0

replace (
	github.com/go-sum/web => ../web
	github.com/go-sum/web/render => ../web/render
)

require (
	github.com/go-playground/validator/v10 v10.30.2
	github.com/go-sum/web/render v0.0.0-00010101000000-000000000000
	golang.org/x/tools v0.43.0
	maragu.dev/gomponents v1.3.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-sum/web v0.0.0-00010101000000-000000000000 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)
