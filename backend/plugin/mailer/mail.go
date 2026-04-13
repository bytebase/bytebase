package mailer

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Sender sends emails.
type Sender interface {
	Send(ctx context.Context, req *SendRequest) error
}

// SendRequest is the request to send an email.
type SendRequest struct {
	To       []string
	Subject  string
	TextBody string
	HTMLBody string
}

// NewSender creates a Sender from the stored email configuration.
func NewSender(cfg *storepb.EmailSetting) (Sender, error) {
	if cfg == nil {
		return nil, errors.Errorf("email setting is nil")
	}
	switch cfg.Type {
	case storepb.EmailSetting_SMTP:
		smtp := cfg.GetSmtp()
		if smtp == nil {
			return nil, errors.Errorf("smtp config is nil")
		}
		return newSMTPSender(cfg.From, cfg.FromName, smtp), nil
	default:
		return nil, errors.Errorf("unsupported email type: %v", cfg.Type)
	}
}
