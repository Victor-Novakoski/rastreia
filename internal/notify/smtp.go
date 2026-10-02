package notify

import (
	"context"
	"time"

	"github.com/wneessen/go-mail"
)

// SMTPConfig points at the mail server. In development it is Mailpit, which
// takes anything without TLS or login.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// TLS requires STARTTLS. Production always sets it.
	TLS bool
}

// SMTP sends e-mails through a mail server.
type SMTP struct {
	cfg SMTPConfig
}

func NewSMTP(cfg SMTPConfig) *SMTP {
	return &SMTP{cfg: cfg}
}

// ErrBadMessage marks an e-mail that will never send, such as an invalid
// address, so the worker parks it instead of retrying.
type ErrBadMessage struct{ Err error }

func (e ErrBadMessage) Error() string { return "bad message: " + e.Err.Error() }
func (e ErrBadMessage) Unwrap() error { return e.Err }

func (s *SMTP) Send(ctx context.Context, e Email) error {
	m := mail.NewMsg()
	if err := m.From(s.cfg.From); err != nil {
		return ErrBadMessage{err}
	}
	if err := m.To(e.To); err != nil {
		return ErrBadMessage{err}
	}
	m.Subject(e.Subject)
	m.SetBodyString(mail.TypeTextPlain, e.Text)
	m.AddAlternativeString(mail.TypeTextHTML, e.HTML)

	opts := []mail.Option{mail.WithPort(s.cfg.Port), mail.WithTimeout(15 * time.Second)}
	if s.cfg.TLS {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}
	if s.cfg.Username != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(s.cfg.Username), mail.WithPassword(s.cfg.Password))
	}
	c, err := mail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return err
	}
	return c.DialAndSendWithContext(ctx, m)
}
