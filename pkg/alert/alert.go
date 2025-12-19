package alert

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/soundprediction/go-predicato/pkg/config"
)

// Alerter defines an interface for sending alerts
type Alerter interface {
	Alert(subject, message string) error
}

// EmailAlerter implements Alerter using SMTP
type EmailAlerter struct {
	cfg config.AlertConfig
}

// NewEmailAlerter creates a new email alerter
func NewEmailAlerter(cfg config.AlertConfig) *EmailAlerter {
	return &EmailAlerter{
		cfg: cfg,
	}
}

// Alert sends an email with the given subject and message
func (a *EmailAlerter) Alert(subject, message string) error {
	if !a.cfg.Enabled {
		return nil
	}

	auth := smtp.PlainAuth("", a.cfg.Username, a.cfg.Password, a.cfg.SMTPHost)

	to := a.cfg.To
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", strings.Join(to, ","), subject, message))

	addr := fmt.Sprintf("%s:%d", a.cfg.SMTPHost, a.cfg.SMTPPort)

	err := smtp.SendMail(addr, auth, a.cfg.From, to, msg)
	if err != nil {
		return fmt.Errorf("failed to send alert email: %w", err)
	}

	return nil
}

// NoOpAlerter is a dummy alerter for when alerting is disabled
type NoOpAlerter struct{}

func (n *NoOpAlerter) Alert(subject, message string) error {
	return nil
}
