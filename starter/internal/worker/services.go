package worker

import (
	"context"
	"log/slog"
	"time"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/features/contact"
	"github.com/go-sum/foundry/internal/common"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/queue/pgstore"
)

func provideServices(ctx context.Context, runtime Runtime) (Services, error) {
	core, err := common.Provide(ctx, runtime.Logger, runtime.Config.DB.DSN, runtime.Config.App.Email)
	if err != nil {
		return Services{}, err
	}
	return Services{
		DBPool:    core.DBPool,
		Processor: newProcessor(core.QueueStore, runtime.Logger, core.EmailSender, runtime.Config.App.Contact),
	}, nil
}

func newProcessor(qStore *pgstore.Store, logger *slog.Logger, emailSender email.Sender, cfg config.ContactConfig) Processor {
	processor := queue.NewProcessor(qStore, queue.WithLogger(logger))
	processor.Register(contact.QueueName, contact.NewNotifyHandler(emailSender, contact.WorkerConfig{
		SendTo:   cfg.SendTo,
		SendFrom: cfg.SendFrom,
	}),
		queue.WithWorkers(2),
		queue.WithMaxAttempts(5),
		queue.WithTimeout(30*time.Second),
	)
	return processor
}
