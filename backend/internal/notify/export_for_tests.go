package notify

// TestConfig mengembalikan konfigurasi SMTP untuk pengujian.
func (s *SMTPSender) TestConfig() SMTPConfig {
	return s.config
}
