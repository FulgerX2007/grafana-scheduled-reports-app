package mail

import (
	"crypto/tls"
	"fmt"
	"io"
	"strings"

	"github.com/yourusername/scheduled-reports-app/pkg/model"
	"gopkg.in/gomail.v2"
)

// Mailer handles email sending
type Mailer struct {
	config model.SMTPConfig
}

// NewMailer creates a new mailer instance
func NewMailer(config model.SMTPConfig) *Mailer {
	return &Mailer{
		config: config,
	}
}

// SendReport sends a report via email
func (m *Mailer) SendReport(recipients model.Recipients, subject, body string, attachment []byte, filename string) error {
	msg := gomail.NewMessage()

	// Set sender
	msg.SetHeader("From", m.config.From)

	// Set recipients
	if len(recipients.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}
	msg.SetHeader("To", recipients.To...)

	if len(recipients.CC) > 0 {
		msg.SetHeader("Cc", recipients.CC...)
	}

	if len(recipients.BCC) > 0 {
		msg.SetHeader("Bcc", recipients.BCC...)
	}

	// Set subject and body
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body)

	// Attach report
	if len(attachment) > 0 {
		msg.Attach(filename, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(attachment)
			return err
		}))
	}

	// Create dialer
	dialer := gomail.NewDialer(m.config.Host, m.config.Port, m.config.Username, m.config.Password)

	if m.config.UseTLS {
		dialer.TLSConfig = &tls.Config{
			InsecureSkipVerify: m.config.SkipTLSVerify,
			ServerName:         m.config.Host,
		}
	} else {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		dialer.SSL = false
	}

	// Send email
	if err := dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// InterpolateTemplate replaces placeholders in the template
func InterpolateTemplate(template string, vars map[string]string) string {
	result := template
	for k, v := range vars {
		placeholder := "{{" + k + "}}"
		result = strings.ReplaceAll(result, placeholder, v)
	}
	return result
}
