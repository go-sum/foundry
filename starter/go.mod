module github.com/go-sum/foundry

go 1.26.0

require (
	github.com/go-sum/assets v0.0.0
	github.com/go-sum/auth v0.0.0
	github.com/go-sum/auth/authui v0.0.0
	github.com/go-sum/auth/pgstore v0.0.0
	github.com/go-sum/auth/provider/pgstore v0.0.0
	github.com/go-sum/componentry v0.0.0
	github.com/go-sum/config v0.0.0
	github.com/go-sum/db v0.0.0
	github.com/go-sum/docs v0.0.0
	github.com/go-sum/kv v0.0.0
	github.com/go-sum/notification v0.0.0
	github.com/go-sum/queue v0.0.0
	github.com/go-sum/queue/pgstore v0.0.0
	github.com/go-sum/showcase v0.0.0
	github.com/go-sum/web v0.0.0
	github.com/go-sum/web/render v0.0.0
	github.com/jackc/pgx/v5 v5.9.1
	go.opentelemetry.io/otel/trace v1.43.0
	gopkg.in/yaml.v3 v3.0.1
	maragu.dev/gomponents v1.3.0
)

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
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
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/schema v1.4.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/redis/go-redis/v9 v9.7.3 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace (
	github.com/go-sum/assets => ../pkg/assets
	github.com/go-sum/auth => ../pkg/auth
	github.com/go-sum/auth/authui => ../pkg/auth/authui
	github.com/go-sum/auth/pgstore => ../pkg/auth/pgstore
	github.com/go-sum/auth/provider/pgstore => ../pkg/auth/provider/pgstore
	github.com/go-sum/componentry => ../pkg/componentry
	github.com/go-sum/config => ../pkg/config
	github.com/go-sum/db => ../pkg/db
	github.com/go-sum/docs => ../pkg/docs
	github.com/go-sum/kv => ../pkg/kv
	github.com/go-sum/notification => ../pkg/notification
	github.com/go-sum/queue => ../pkg/queue
	github.com/go-sum/queue/pgstore => ../pkg/queue/pgstore
	github.com/go-sum/showcase => ../pkg/showcase
	github.com/go-sum/web => ../pkg/web
	github.com/go-sum/web/render => ../pkg/web/render
)
