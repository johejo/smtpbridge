package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/emersion/go-smtp"
	"github.com/tailscale/tscert"
	"golang.org/x/net/html"
)

var (
	tlsMethod         string
	tlsCertPemFile    string
	tlsKeyPemFile     string
	providerSelection string
	addr              string
)

func init() {
	flag.StringVar(&tlsMethod, "tls-method", "key-pair-file", "tls config method: available values are key-pair-file and tscert. tscert is available only when running on tailscale.")
	flag.StringVar(&tlsCertPemFile, "tls-cert-pem-file", "", "tls cert pem file: available when tls-method is key-pair-file")
	flag.StringVar(&tlsKeyPemFile, "tls-key-pem-file", "", "tls key pem file: available when tls-method is key-pair-file")
	flag.StringVar(&addr, "addr", ":1587", "listen address")
	flag.StringVar(&providerSelection, "provider-selection", "random", "provider selection method for multiple providers: available values are random and round-robin")
}

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	ctx := context.Background()

	var tlsCfg *tls.Config
	switch tlsMethod {
	case "key-pair-file":
		cert, err := tls.LoadX509KeyPair(tlsCertPemFile, tlsKeyPemFile)
		if err != nil {
			log.Fatal(err)
		}
		tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	case "tscert":
		status, err := tscert.GetStatus(ctx)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("tscert status: %+v", status)
		tlsCfg = &tls.Config{
			GetCertificate: tscert.GetCertificate,
		}
	default:
		log.Fatalf("invalid tls method: %s", tlsMethod)
	}

	var auth *authPlain = nil
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	if username != "" && password != "" {
		auth = &authPlain{username: username, password: password}
	}

	var backends []smtp.Backend
	if resendApiKey := os.Getenv("RESEND_API_KEY"); resendApiKey != "" {
		log.Printf("resend backend enabled")
		backends = append(backends, newResendBackend(resendApiKey, auth))
	}
	if sendgridApiKey := os.Getenv("SENDGRID_API_KEY"); sendgridApiKey != "" {
		log.Printf("sendgrid backend enabled")
		backends = append(backends, newSendGridBackend(sendgridApiKey, auth))
	}
	if len(backends) == 0 {
		log.Fatal("no backends enabled")
	}

	var backend smtp.Backend
	switch providerSelection {
	case "random":
		backend = &randomBackend{backends: backends}
	case "round-robin":
		backend = &roundRobinBackend{backends: backends}
	default:
		log.Fatalf("invalid provider selection: %s", providerSelection)
	}

	s := smtp.NewServer(backend)
	s.TLSConfig = tlsCfg
	s.Addr = addr

	log.Println(s.ListenAndServeTLS())
}

type randomBackend struct {
	backends []smtp.Backend
}

func (b *randomBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	if len(b.backends) == 1 {
		return b.backends[0].NewSession(c)
	}
	return b.backends[rand.Intn(len(b.backends))].NewSession(c)
}

type roundRobinBackend struct {
	backends []smtp.Backend
	index    int
	mu       sync.Mutex
}

func (b *roundRobinBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	if len(b.backends) == 1 {
		return b.backends[0].NewSession(c)
	}
	b.mu.Lock()
	defer func() {
		if b.index == len(b.backends)-1 {
			b.index = 0
		} else {
			b.index++
		}
		b.mu.Unlock()
	}()
	return b.backends[b.index].NewSession(c)
}

const (
	subjectPrefix = "Subject: "
	fromPrefix    = "From: "
	ccPrefix      = "Cc: "
	toPrefix      = "To: "
	replyToPrefix = "Reply-To: "
)

type parsedData struct {
	Subject string
	From    string
	ReplyTo string
	To      []string
	Cc      []string
	Text    string
	Html    string
}

func isHTML(s string) bool {
	_, err := html.Parse(strings.NewReader(s))
	return err == nil
}

func parseData(r io.Reader) (*parsedData, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	log.Printf("data: %s", string(b))

	tmp := bytes.Split(b, []byte("\r\n"))
	data := make([]string, 0, len(tmp))
	for _, v := range tmp {
		if len(v) == 0 {
			continue
		}
		data = append(data, string(v))
	}
	log.Printf("data: %s", strings.Join(data, ","))

	var parsed parsedData
	for _, v := range data {
		if subject, ok := strings.CutPrefix(v, subjectPrefix); ok {
			parsed.Subject = subject
			continue
		}
		if from, ok := strings.CutPrefix(v, fromPrefix); ok {
			parsed.From = from
			continue
		}
		if replyTo, ok := strings.CutPrefix(v, replyToPrefix); ok {
			parsed.ReplyTo = replyTo
			continue
		}
		if to, ok := strings.CutPrefix(v, toPrefix); ok {
			parsed.To = append(parsed.To, to)
			continue
		}
		if cc, ok := strings.CutPrefix(v, ccPrefix); ok {
			parsed.Cc = append(parsed.Cc, cc)
			continue
		}
	}
	body := data[len(data)-1]
	if isHTML(body) {
		parsed.Html = body
	} else {
		parsed.Text = body
	}
	log.Printf("parsedData: %+v", parsed)

	return &parsed, nil
}

type authPlain struct {
	username string
	password string
}
