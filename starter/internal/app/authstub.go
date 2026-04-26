package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-sum/auth"
	"github.com/go-sum/foundry/config"
)

// authLogNotifier logs verification codes to the application logger instead of
// sending emails. It must never be used in production — use mustNotProductionLogNotifier
// to enforce this at startup.
type authLogNotifier struct {
	logger *slog.Logger
}

func (n *authLogNotifier) SendVerification(_ context.Context, input auth.DeliveryInput) error {
	n.logger.Info("auth verification code",
		"purpose", input.Purpose,
		"email", input.Email,
		"code", input.Code,
		"verify_url", input.VerifyURL,
	)
	return nil
}

// mustNotProductionLogNotifier returns an authLogNotifier for non-production
// environments. It returns a startup error in production because logging raw
// TOTP codes and verify URLs is not safe in a production log aggregator.
// Replace this with a real email-delivering Notifier for production use.
func mustNotProductionLogNotifier(env config.Env, logger *slog.Logger) (auth.Notifier, error) {
	if env == config.Production {
		return nil, fmt.Errorf("authLogNotifier must not be used in production; implement auth.Notifier with real email delivery")
	}
	return &authLogNotifier{logger: logger}, nil
}
