package showcase

import (
	"github.com/go-sum/foundry/pkg/componentry/icons"
	kvstore "github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/showcase/base"
	"github.com/go-sum/foundry/pkg/showcase/componentry"
	showcasedb "github.com/go-sum/foundry/pkg/showcase/db"
	showcasekv "github.com/go-sum/foundry/pkg/showcase/kv"
	showcasequeue "github.com/go-sum/foundry/pkg/showcase/queue"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PageFunc = base.PageFunc

// Config configures the full showcase route tree.
type Config struct {
	Icons *icons.Registry
	DB    *pgxpool.Pool
	KV    kvstore.Store
	Page  PageFunc
}

// Routes returns the package-owned showcase route tree.
func Routes(cfg Config) []router.Node {
	componentryCfg := componentry.DefaultConfig()
	componentryCfg.Icons = cfg.Icons
	componentryCfg.Page = cfg.Page

	nodes := componentry.Routes(componentryCfg)

	if cfg.DB != nil {
		dbCfg := showcasedb.DefaultConfig()
		dbCfg.Pool = cfg.DB
		dbCfg.Page = cfg.Page
		nodes = append(nodes, showcasedb.Routes(dbCfg)...)

		queueCfg := showcasequeue.DefaultConfig()
		queueCfg.Pool = cfg.DB
		queueCfg.Page = cfg.Page
		nodes = append(nodes, showcasequeue.Routes(queueCfg)...)
	}

	if cfg.KV != nil {
		kvCfg := showcasekv.DefaultConfig()
		kvCfg.Store = cfg.KV
		kvCfg.Page = cfg.Page
		nodes = append(nodes, showcasekv.Routes(kvCfg)...)
	}

	return nodes
}
