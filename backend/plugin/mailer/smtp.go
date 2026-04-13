package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const (
	connectTimeout = 10 * time.Second
)

type smtpSender struct {
	from     string
	fromName string
	config   *storepb.EmailSetting_SMTPConfig
}

func newSMTPSender(from, fromName string, config *storepb.EmailSetting_SMTPConfig) *smtpSender {
	return &smtpSender{from: from, fromName: fromName, config: config}
}

func (s *smtpSender) Send(_ context.Context, req *SendRequest) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	client, err := s.dial(addr)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to SMTP server %s", addr)
	}
	defer client.Close()

	if s.config.Encryption == storepb.EmailSetting_SMTPConfig_STARTTLS {
		tlsConfig := &tls.Config{ServerName: s.config.Host}
		if err := client.StartTLS(tlsConfig); err != nil {
			return errors.Wrap(err, "STARTTLS failed")
		}
	}

	if auth := s.auth(); auth != nil {
		if err := client.Auth(auth); err != nil {
			return errors.Wrap(err, "SMTP authentication failed")
		}
	}

	if err := client.Mail(s.from); err != nil {
		return errors.Wrap(err, "MAIL FROM failed")
	}
	for _, to := range req.To {
		if err := client.Rcpt(to); err != nil {
			return errors.Wrapf(err, "RCPT TO %s failed", to)
		}
	}

	w, err := client.Data()
	if err != nil {
		return errors.Wrap(err, "DATA command failed")
	}
	msg := s.buildMessage(req)
	if _, err := w.Write([]byte(msg)); err != nil {
		return errors.Wrap(err, "failed to write message")
	}
	if err := w.Close(); err != nil {
		return errors.Wrap(err, "failed to close data writer")
	}

	return client.Quit()
}

func (s *smtpSender) dial(addr string) (*smtp.Client, error) {
	switch s.config.Encryption {
	case storepb.EmailSetting_SMTPConfig_SSL_TLS:
		tlsConfig := &tls.Config{ServerName: s.config.Host}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: connectTimeout}, "tcp", addr, tlsConfig)
		if err != nil {
			return nil, err
		}
		return smtp.NewClient(conn, s.config.Host)
	default:
		conn, err := net.DialTimeout("tcp", addr, connectTimeout)
		if err != nil {
			return nil, err
		}
		return smtp.NewClient(conn, s.config.Host)
	}
}

func (s *smtpSender) auth() smtp.Auth {
	if s.config.Authentication == storepb.EmailSetting_SMTPConfig_AUTHENTICATION_NONE {
		return nil
	}
	if s.config.Username == "" && s.config.Password == "" {
		return nil
	}
	switch s.config.Authentication {
	case storepb.EmailSetting_SMTPConfig_CRAM_MD5:
		return smtp.CRAMMD5Auth(s.config.Username, s.config.Password)
	default:
		// PLAIN, LOGIN, and UNSPECIFIED all use PlainAuth.
		return smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}
}

func (s *smtpSender) buildMessage(req *SendRequest) string {
	var b strings.Builder

	fromHeader := s.from
	if s.fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", s.fromName), s.from)
	}
	fmt.Fprintf(&b, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(req.To, ", "))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", req.Subject))
	fmt.Fprint(&b, "MIME-Version: 1.0\r\n")

	if req.HTMLBody != "" {
		boundary := "bytebase-email-boundary"
		fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary)
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		fmt.Fprint(&b, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
		fmt.Fprintf(&b, "%s\r\n", req.TextBody)
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		fmt.Fprint(&b, "Content-Type: text/html; charset=utf-8\r\n\r\n")
		fmt.Fprintf(&b, "%s\r\n", req.HTMLBody)
		fmt.Fprintf(&b, "--%s--\r\n", boundary)
	} else {
		fmt.Fprint(&b, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
		fmt.Fprintf(&b, "%s\r\n", req.TextBody)
	}

	return b.String()
}
