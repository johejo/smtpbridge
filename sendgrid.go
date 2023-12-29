package main

import (
	"fmt"
	"io"
	"log"
	netmail "net/mail"

	"github.com/emersion/go-smtp"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type sendgridBackend struct {
	auth   *authPlain
	client *sendgrid.Client
}

func newSendGridBackend(apiKey string, auth *authPlain) *sendgridBackend {
	return &sendgridBackend{
		client: sendgrid.NewSendClient(apiKey),
		auth: auth,
	}
}

func (b *sendgridBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &sendgridSession{client: b.client, auth: b.auth}, nil
}

type sendgridSession struct {
	auth   *authPlain
	client *sendgrid.Client
}

func (s *sendgridSession) Mail(from string, opts *smtp.MailOptions) error {
	return nil
}

func (s *sendgridSession) Rcpt(to string, opts *smtp.RcptOptions) error {
	return nil
}

func (s *sendgridSession) Reset() {
}

func (s *sendgridSession) Logout() error {
	return nil
}

func (s *sendgridSession) AuthPlain(username, password string) error {
	if s.auth == nil {
		return nil
	}
	// TODO: support hashed passwords?
	if s.auth.username != username || s.auth.password != password {
		return fmt.Errorf("invalid username or password")
	}
	return nil
}

func (s *sendgridSession) Data(r io.Reader) error {
	d, err := parseData(r)
	if err != nil {
		log.Println(err)
		return err
	}
	email := mail.NewV3Mail()
	from, err := netmail.ParseAddress(d.From)
	if err != nil {
		log.Println(err)
		return err
	}
	email.From = mail.NewEmail(from.Name, from.Address)
	if d.ReplyTo != "" {
		replyTo, err := netmail.ParseAddress(d.ReplyTo)
		if err != nil {
			log.Println(err)
			return err
		}
		email.ReplyTo = mail.NewEmail(replyTo.Name, replyTo.Address)
	}
	email.Subject = d.Subject

	p := mail.NewPersonalization()
	for _, _to := range d.To {
		to, err := netmail.ParseAddress(_to)
		if err != nil {
			log.Println(err)
			return err
		}
		p.AddTos(mail.NewEmail(to.Name, to.Address))
	}
	for _, _cc := range d.Cc {
		cc, err := netmail.ParseAddress(_cc)
		if err != nil {
			log.Println(err)
			return err
		}
		p.AddCCs(mail.NewEmail(cc.Name, cc.Address))
	}
	email.Personalizations = []*mail.Personalization{p}
	if d.Text != "" {
		email.AddContent(mail.NewContent("text/plain", d.Text))
	}
	if d.Html != "" {
		email.AddContent(mail.NewContent("text/html", d.Html))
	}

	resp, err := s.client.Send(email)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("resp: %+v", resp)
	return nil
}
