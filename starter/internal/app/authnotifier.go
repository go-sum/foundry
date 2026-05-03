package app

import (
	"context"
	"fmt"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/notification/email"
)

type emailNotifier struct {
	sender email.Sender
	from   string
}

func newEmailNotifier(sender email.Sender, from string) auth.Notifier {
	return &emailNotifier{sender: sender, from: from}
}

func (n *emailNotifier) SendVerification(ctx context.Context, input auth.DeliveryInput) error {
	subject := verificationSubject(input.Purpose)
	var text string
	if input.Code != "" {
		text = fmt.Sprintf("Your verification code is: %s\n", input.Code)
	}
	if input.VerifyURL != "" {
		text += fmt.Sprintf("Verify: %s\n", input.VerifyURL)
	}
	return n.sender.Send(ctx, email.Message{
		To:      input.Email,
		From:    n.from,
		Subject: subject,
		Text:    text,
	})
}

func verificationSubject(purpose auth.FlowPurpose) string {
	switch purpose {
	case auth.FlowSignup:
		return "Verify your account"
	case auth.FlowSignin:
		return "Sign in verification"
	case auth.FlowEmailChange:
		return "Verify your new email"
	case auth.FlowAlreadyRegistered:
		return "Account already exists"
	default:
		return "Verification"
	}
}
