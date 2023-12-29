package main

import (
	"fmt"
	"io"
	"log"

	"github.com/emersion/go-smtp"
	"github.com/resendlabs/resend-go/v2"
)

type resendBackend struct {
	auth   *authPlain
	client *resend.Client
}

func newResendBackend(apiKey string, auth *authPlain) *resendBackend {
	return &resendBackend{
		client: resend.NewClient(apiKey),
		auth:   auth,
	}
}

func (b *resendBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &resendSession{client: b.client, auth: b.auth}, nil
}

type resendSession struct {
	auth   *authPlain
	client *resend.Client
}

func (s *resendSession) Mail(from string, opts *smtp.MailOptions) error {
	return nil
}

func (s *resendSession) Rcpt(to string, opts *smtp.RcptOptions) error {
	return nil
}

func (s *resendSession) Reset() {
}

func (s *resendSession) Logout() error {
	return nil
}

func (s *resendSession) AuthPlain(username, password string) error {
	if s.auth == nil {
		return nil
	}
	// TODO: support hashed passwords?
	if s.auth.username != username || s.auth.password != password {
		return fmt.Errorf("invalid username or password")
	}
	return nil
}

func (s *resendSession) Data(r io.Reader) error {
	d, err := parseData(r)
	if err != nil {
		return err
	}
	req := &resend.SendEmailRequest{
		From:    d.From,
		To:      d.To,
		Subject: d.Subject,
		Cc:      d.Cc,
		ReplyTo: d.ReplyTo,
		// TODO attachments, headers
	}
	if d.Html != "" {
		req.Html = d.Html
	}
	if d.Text != "" {
		req.Text = d.Text
	}

	resp, err := s.client.Emails.Send(req)
	if err != nil {
		return err
	}
	log.Printf("resp: %+v", resp)
	return nil
}
