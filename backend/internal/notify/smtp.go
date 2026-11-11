package notify

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Sender mengirim satu pemberitahuan.
type Sender interface {
	Send(ctx context.Context, message Message) error
}

// Transport mengirim email mentah ke server SMTP.
type Transport interface {
	Send(ctx context.Context, raw []byte, from string, recipients []string) error
}

// SMTPConfig memuat konfigurasi koneksi SMTP.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// UseTLS mengaktifkan TLS saat koneksi. STARTTLS dipakai bila server
	// hanya mendukungnya, bergantung pada kebijakan provider.
	UseTLS bool
	// Timeout membatasi waktu koneksi dan pengiriman.
	Timeout time.Duration
}

// SMTPSender mengirim email melalui server SMTP.
type SMTPSender struct {
	config    SMTPConfig
	transport Transport
}

// NewSMTPSender membuat pengirim email SMTP.
func NewSMTPSender(config SMTPConfig) *SMTPSender {
	if config.Timeout <= 0 {
		config.Timeout = 15 * time.Second
	}
	return &SMTPSender{config: config, transport: &smtpTransport{config: config}}
}

// NewSMTPSenderWithTransport membuat pengirim dengan transport khusus,
// dipakai pada pengujian.
func NewSMTPSenderWithTransport(config SMTPConfig, transport Transport) *SMTPSender {
	sender := NewSMTPSender(config)
	if transport != nil {
		sender.transport = transport
	}
	return sender
}

// Send mengirim satu email.
func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	if err := message.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(s.config.From) == "" {
		return errors.New("smtp sender address is not configured")
	}
	raw := buildMessage(s.config.From, message)
	return s.transport.Send(ctx, raw, s.config.From, []string{message.To})
}

// buildMessage menyusun email mentah dengan header wajib.
func buildMessage(from string, message Message) []byte {
	var builder strings.Builder
	builder.WriteString("From: " + from + "\r\n")
	builder.WriteString("To: " + message.To + "\r\n")
	builder.WriteString("Subject: " + message.Subject + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(strings.ReplaceAll(message.Body, "\n", "\r\n"))
	builder.WriteString("\r\n")
	return []byte(builder.String())
}

// smtpTransport melakukan pengiriman melalui server SMTP.
type smtpTransport struct {
	config SMTPConfig
}

func (t *smtpTransport) Send(ctx context.Context, raw []byte, from string, recipients []string) error {
	address := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	dialer := &net.Dialer{Timeout: t.config.Timeout}

	var conn net.Conn
	var err error
	if t.config.UseTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{ServerName: t.config.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return fmt.Errorf("connect smtp: %w", err)
	}

	client, err := smtp.NewClient(conn, t.config.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("start smtp session: %w", err)
	}
	defer client.Close()

	// Batas waktu ditetapkan pada koneksi agar operasi baca/tulis SMTP
	// tidak menggantung.
	deadline := time.Now().Add(t.config.Timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)

	if t.config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", t.config.Username, t.config.Password, t.config.Host)); err != nil {
			return fmt.Errorf("authenticate smtp: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail command: %w", err)
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp rcpt command: %w", err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data command: %w", err)
	}
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write smtp body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp body: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}
