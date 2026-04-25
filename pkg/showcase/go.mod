module github.com/go-sum/showcase

go 1.26.0

replace (
	github.com/go-sum/componentry => ../componentry
	github.com/go-sum/kv => ../kv
	github.com/go-sum/web => ../web
	github.com/go-sum/web/render => ../web/render
)

require (
	github.com/go-sum/componentry v0.0.0
	github.com/go-sum/kv v0.0.0
	github.com/go-sum/web v0.0.0
	github.com/go-sum/web/render v0.0.0
	github.com/jackc/pgx/v5 v5.9.1
	maragu.dev/gomponents v1.3.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
)
