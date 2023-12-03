// Package mail is a mail delivery plugin based on SMTP.
package mail

// Usage:
//  email := NewEmailMsg().SetFrom("Bytebase <from@bytebase.com>").AddTo("Customer <to@bytebase.com>").SetSubject("Test Email Subject").SetBody(`
// <!DOCTYPE html>
// <html>
// <head>
// 	<title>HTML Test</title>
// </head>
// <body>
// 	<h1>This is a mail delivery test.</h1>
// </body>
// </html>
// 	`)
// 	fmt.Printf("email: %+v\n", email)
// 	client := NewSMTPClient("smtp.gmail.com", 587)
// 	client.SetAuthType(SMTPAuthTypePlain)
// 	client.SetAuthCredentials("from@bytebase.com", "nqxxxxxxxxxxxxxx")
// 	client.SetEncryptionType(SMTPEncryptionTypeSTARTTLS)
// 	if err := client.SendMail(email); err != nil {
// 		t.Fatalf("SendMail failed: %v", err)
// 	}

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/mail"
	"net/smtp"

	"github.com/jordan-wright/email"
	"github.com/pkg/errors"
)

// Email is the email to be sent.
type Email struct {
	err     error
	from    string
	subject string

	e *email.Email
}

// NewEmailMsg returns a new email message.
func NewEmailMsg() *Email {
	e := &Email{
		e: email.NewEmail(),
	}
	return e
}

// SetFrom sets the from address of the SMTP client.
// Only accept the valid RFC 5322 address, e.g. "Bytebase <support@bytebase.com>".
func (e *Email) SetFrom(from string) *Email {
	if e.err != nil {
		return e
	}
	if e.from != "" {
		e.err = errors.New("From address already set")
		return e
	}

	parsedAddr, err := mail.ParseAddress(from)
	if err != nil {
		e.err = errors.Wrapf(err, "Invalid from address: %s", from)
	}
	e.from = parsedAddr.Address
	e.e.From = parsedAddr.String()
	return e
}

// AddTo adds the to address of the SMTP client.
// Only accept the valid RFC 5322 address, e.g. "Bytebase <support@bytebase.com>".
func (e *Email) AddTo(to ...string) *Email {
	if e.err != nil {
		return e
	}
	var buf []*mail.Address
	for _, toAddress := range to {
		parsedAddr, err := mail.ParseAddress(toAddress)
		if err != nil {
			e.err = errors.Wrapf(err, "Invalid to address: %s", toAddress)
			return e
		}
		buf = append(buf, parsedAddr)
	}
	for _, addr := range buf {
		e.e.To = append(e.e.To, addr.String())
	}
	return e
}

// SetSubject sets the subject of the SMTP client.
func (e *Email) SetSubject(subject string) *Email {
	if e.err != nil {
		return e
	}
	if e.subject != "" {
		e.err = errors.New("Subject already set")
		return e
	}
	e.subject = subject
	e.e.Subject = subject
	return e
}

// SetBody sets the body of the SMTP client. It must be html formatted.
func (e *Email) SetBody(body string) *Email {
	e.e.HTML = []byte(body)
	return e
}

// The ContentType is the type of the content.
// https://cloud.google.com/appengine/docs/legacy/standard/php/mail/mail-with-headers-attachments.
type ContentType string

const (
	// ContentTypeImagePNG is the content type of the file with png extension.
	ContentTypeImagePNG ContentType = "image/png"
)

// Attach attaches the file to the email, and returns the filename of the attachment.
// Caller can use filename as content id to reference the attachment in the email body.
func (e *Email) Attach(reader io.Reader, filename string, contentType ContentType) (string, error) {
	attachment, err := e.e.Attach(reader, filename, string(contentType))
	if err != nil {
		return "", err
	}
	return attachment.Filename, nil
}

// SMTPAuthType is the type of SMTP authentication.
type SMTPAuthType uint

const (
	// SMTPAuthTypeNone is the NONE auth type of SMTP.
	SMTPAuthTypeNone = iota
	// SMTPAuthTypePlain is the PLAIN auth type of SMTP.
	SMTPAuthTypePlain
	// SMTPAuthTypeLogin is the LOGIN auth type of SMTP.
	SMTPAuthTypeLogin
	// SMTPAuthTypeCRAMMD5 is the CRAM-MD5 auth type of SMTP.
	SMTPAuthTypeCRAMMD5
)

// SMTPEncryptionType is the type of SMTP encryption.
type SMTPEncryptionType uint

const (
	// SMTPEncryptionTypeNone is the NONE encrypt type of SMTP.
	SMTPEncryptionTypeNone = iota
	// SMTPEncryptionTypeSSLTLS is the SSL/TLS encrypt type of SMTP.
	SMTPEncryptionTypeSSLTLS
	// SMTPEncryptionTypeSTARTTLS is the STARTTLS encrypt type of SMTP.
	SMTPEncryptionTypeSTARTTLS
)

// SMTPClient is the client of SMTP.
type SMTPClient struct {
	host           string
	port           int
	authType       SMTPAuthType
	username       string
	password       string
	encryptionType SMTPEncryptionType
}

// NewSMTPClient returns a new SMTP client.
func NewSMTPClient(host string, port int) *SMTPClient {
	return &SMTPClient{
		host:           host,
		port:           port,
		authType:       SMTPAuthTypeNone,
		username:       "",
		password:       "",
		encryptionType: SMTPEncryptionTypeNone,
	}
}

// SendMail sends the email.
func (c *SMTPClient) SendMail(e *Email) error {
	if e.err != nil {
		return e.err
	}

	switch c.encryptionType {
	case SMTPEncryptionTypeNone:
		return e.e.Send(fmt.Sprintf("%s:%d", c.host, c.port), c.getAuth())
	case SMTPEncryptionTypeSSLTLS:
		return e.e.SendWithTLS(fmt.Sprintf("%s:%d", c.host, c.port), c.getAuth(), &tls.Config{ServerName: c.host})
	case SMTPEncryptionTypeSTARTTLS:
		return e.e.SendWithStartTLS(fmt.Sprintf("%s:%d", c.host, c.port), c.getAuth(), &tls.Config{InsecureSkipVerify: true})
	default:
		return errors.Errorf("Unknown SMTP encryption type: %d", c.encryptionType)
	}
}

// SetAuthType sets the auth type of the SMTP client.
func (c *SMTPClient) SetAuthType(authType SMTPAuthType) *SMTPClient {
	c.authType = authType
	return c
}

// SetAuthCredentials sets the auth credentials of the SMTP client.
func (c *SMTPClient) SetAuthCredentials(username, password string) *SMTPClient {
	c.username = username
	c.password = password
	return c
}

func (c *SMTPClient) getAuth() smtp.Auth {
	switch c.authType {
	case SMTPAuthTypeNone:
		return nil
	case SMTPAuthTypePlain:
		return smtp.PlainAuth("", c.username, c.password, c.host)
	case SMTPAuthTypeLogin:
		return LoginAuth(c.username, c.password)
	case SMTPAuthTypeCRAMMD5:
		return smtp.CRAMMD5Auth(c.username, c.password)
	default:
		return nil
	}
}

// SetEncryptionType sets the encryption type of the SMTP client.
func (c *SMTPClient) SetEncryptionType(encryptionType SMTPEncryptionType) {
	c.encryptionType = encryptionType
}
