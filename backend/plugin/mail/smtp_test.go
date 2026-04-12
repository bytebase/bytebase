package mail

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestBuildMessage_PlainText(t *testing.T) {
	s := &smtpSender{from: "noreply@example.com", fromName: "Bytebase"}
	msg := s.buildMessage(&SendRequest{
		To:       []string{"user@example.com"},
		Subject:  "Test Subject",
		TextBody: "Hello, World!",
	})
	assert.Contains(t, msg, "From: Bytebase <noreply@example.com>")
	assert.Contains(t, msg, "To: user@example.com")
	assert.Contains(t, msg, "Subject: Test Subject")
	assert.Contains(t, msg, "Content-Type: text/plain; charset=utf-8")
	assert.Contains(t, msg, "Hello, World!")
	assert.NotContains(t, msg, "multipart")
}

func TestBuildMessage_HTML(t *testing.T) {
	s := &smtpSender{from: "noreply@example.com", fromName: ""}
	msg := s.buildMessage(&SendRequest{
		To:       []string{"a@b.com", "c@d.com"},
		Subject:  "HTML Test",
		TextBody: "plain text",
		HTMLBody: "<p>html body</p>",
	})
	assert.Contains(t, msg, "From: noreply@example.com")
	assert.Contains(t, msg, "To: a@b.com, c@d.com")
	assert.Contains(t, msg, "multipart/alternative")
	assert.Contains(t, msg, "plain text")
	assert.Contains(t, msg, "<p>html body</p>")
}

func TestNewSender_SMTP(t *testing.T) {
	sender, err := NewSender(&storepb.EmailSetting{
		From: "test@example.com",
		Type: storepb.EmailSetting_SMTP,
		Config: &storepb.EmailSetting_Smtp{
			Smtp: &storepb.EmailSetting_SMTPConfig{
				Host:       "localhost",
				Port:       25,
				Encryption: storepb.EmailSetting_SMTPConfig_ENCRYPTION_NONE,
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, sender)
}

func TestNewSender_NilConfig(t *testing.T) {
	_, err := NewSender(nil)
	assert.Error(t, err)
}

func TestNewSender_UnsupportedType(t *testing.T) {
	_, err := NewSender(&storepb.EmailSetting{
		From: "test@example.com",
		Type: storepb.EmailSetting_TYPE_UNSPECIFIED,
	})
	assert.Error(t, err)
}

func TestSMTPSend_ConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	sender, err := NewSender(&storepb.EmailSetting{
		From: "test@example.com",
		Type: storepb.EmailSetting_SMTP,
		Config: &storepb.EmailSetting_Smtp{
			Smtp: &storepb.EmailSetting_SMTPConfig{
				Host:       "127.0.0.1",
				Port:       int32(port),
				Encryption: storepb.EmailSetting_SMTPConfig_ENCRYPTION_NONE,
			},
		},
	})
	require.NoError(t, err)

	err = sender.Send(context.Background(), &SendRequest{
		To:       []string{"user@example.com"},
		Subject:  "Test",
		TextBody: "body",
	})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "connect") || strings.Contains(err.Error(), "refused"))
}

func TestAuth_None(t *testing.T) {
	s := &smtpSender{config: &storepb.EmailSetting_SMTPConfig{
		Authentication: storepb.EmailSetting_SMTPConfig_AUTHENTICATION_NONE,
	}}
	assert.Nil(t, s.auth())
}

func TestAuth_Plain(t *testing.T) {
	s := &smtpSender{config: &storepb.EmailSetting_SMTPConfig{
		Authentication: storepb.EmailSetting_SMTPConfig_PLAIN,
		Username:       "user",
		Password:       "pass",
		Host:           "smtp.example.com",
	}}
	assert.NotNil(t, s.auth())
}
