# smtpbridge

smtpbridge is a bridge tool between SMTP Client and HTTP API Based Email Provider.

```
SMTP Client <-> smtpbridge <-> HTTP API Based Email Provider
```

## Supported Email Providers

- SendGrid
- Resend

## Installation

```
go install github.com/johejo/smtpbridge@latest
```

## Provider Configuration

Environment Variables

### SendGrid

- `SENDGRID_API_KEY`

### Resend

- `RESEND_API_KEY`

## SMTP Authentication

Specify `SMTP_USERNAME` and `SMTP_PASSWORD` via environment variables.

## TLS

smtpbridge only supports SMPT over TLS, so does not support plain-text SMTP.

### Key Pair

Specify pem file path via command line flags.

### Tailscale TLS Certificate

TLS certificates are easily available if tailscale is running on the host.

https://tailscale.com/kb/1153/enabling-https

Just specify flag `-tls-method=tscert`

## Command Line Flags

```
Usage of smtpbridge:
  -addr string
        listen address (default ":1587")
  -provider-selection string
        provider selection method for multipe providers: available values are random and round-robin (default "random")
  -tls-cert-pem-file string
        tls cert pem file: available when tls-method is key-pair-file
  -tls-key-pem-file string
        tls key pem file: available when tls-method is key-pair-file
  -tls-method string
        tls config method: available values are key-pair-file and tscert. tscert is available only when running on tailscale. (default "key-pair-file")
```
