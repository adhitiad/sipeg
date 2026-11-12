package notify

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// smtpServer adalah server SMTP uji yang menawarankan STARTTLS.
type smtpServer struct {
	listener net.Listener
	cert     tls.Certificate
	// received holds the contents of the message body.
	received []string
	// startTLSOffered records whether the client issued the STARTTLS command.
	startTLSOffered bool
	mu              sync.Mutex
	done            chan struct{}
}

// startSMTPServer menjalankan server SMTP lokal dengan dukungan STARTTLS.
//
// Server dipakai untuk memastikan jalur pengiriman memakai TLS sebelum
// autentikasi. Server submission pada porta 587 behaves like this: the
// connection starts in plain text and is upgraded via STARTTLS.
func startSMTPServer(t *testing.T) *smtpServer {
	t.Helper()
	cert := generateTestCertificate(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &smtpServer{listener: listener, cert: cert, done: make(chan struct{})}
	go server.serve()
	t.Cleanup(func() {
		_ = listener.Close()
		<-server.done
	})
	return server
}

// serve menjalankan satu percakapan SMTP.
func (s *smtpServer) serve() {
	defer close(s.done)
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	write := func(format string, args ...any) {
		fmt.Fprintf(writer, format, args...)
		_ = writer.Flush()
	}

	write("220 smtp.test ESMTP siap\r\n")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"):
			write("250-smtp.test\r\n250-STARTTLS\r\n250 SIZE 10485760\r\n")
		case strings.HasPrefix(command, "HELO"):
			write("250 smtp.test\r\n")
		case strings.HasPrefix(command, "STARTTLS"):
			s.mu.Lock()
			s.startTLSOffered = true
			s.mu.Unlock()
			write("220 Siap memulai TLS\r\n")
			tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{s.cert}})
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
			writer = bufio.NewWriter(conn)
		case strings.HasPrefix(command, "AUTH"):
			write("235 2.7.0 Autentikasi berhasil\r\n")
		case strings.HasPrefix(command, "MAIL"), strings.HasPrefix(command, "RCPT"):
			write("250 2.1.0 Ok\r\n")
		case strings.HasPrefix(command, "DATA"):
			write("354 Akhiri data dengan <CR><LF>.<CR><LF>\r\n")
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
				s.mu.Lock()
				s.received = append(s.received, strings.TrimRight(dataLine, "\r\n"))
				s.mu.Unlock()
			}
			write("250 2.0.0 Ok: antrean\r\n")
		case strings.HasPrefix(command, "QUIT"):
			write("221 2.0.0 Bye\r\n")
			return
		default:
			write("250 2.0.0 Ok\r\n")
		}
	}
}

// body mengembalikan isi pesan yang diterima server.
func (s *smtpServer) body() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.received, "\n")
}

// offeredStartTLS melapor apakah client mengirim perintah STARTTLS.
func (s *smtpServer) offeredStartTLS() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startTLSOffered
}

// generateTestCertificate membuat sertifikat sendiri untuk server uji.
func generateTestCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "smtp.test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"smtp.test", "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	raw, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{raw}, PrivateKey: key}
}

// TestSMTPTransportUpgradesToTLSWhenServerOffersStartTLS memastikan jalur
// pengiriman memakai TLS sebelum autentikasi.
//
// Server submission pada porta 587 hanya menerima koneksi polos lalu
// menawarkan STARTTLS. Tanpa peningkatan otomatis, kredensial dikirim lewat
// koneksi yang belum dienkripsi atau percakapan gagal sama sekali.
func TestSMTPTransportUpgradesToTLSWhenServerOffersStartTLS(t *testing.T) {
	server := startSMTPServer(t)
	host, port, err := net.SplitHostPort(server.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	leaf, err := x509.ParseCertificate(server.cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	pool.AddCert(leaf)

	sender := NewSMTPSender(SMTPConfig{
		Host:     host,
		Port:     atoi(port),
		Username: "penguji@example.test",
		Password: "rahasia",
		From:     "noreply@example.test",
		UseTLS:   false,
		TLSConfig: &tls.Config{
			RootCAs:    pool,
			ServerName: "smtp.test",
			MinVersion: tls.VersionTLS12,
		},
		Timeout: 10 * time.Second,
	})

	err = sender.Send(context.Background(), GenericNotice(
		"pegawai@example.test", "Halo", "Isi pesan", "notif-starttls"))
	if err != nil {
		t.Fatalf("pengiriman gagal: %v", err)
	}
	if !server.offeredStartTLS() {
		t.Fatal("client tidak mengirim perintah STARTTLS")
	}
	if !strings.Contains(server.body(), "Isi pesan") {
		t.Fatalf("isi pesan tidak diterima:\n%s", server.body())
	}
	if !strings.Contains(server.body(), "From: noreply@example.test") {
		t.Fatalf("header pengirim tidak diterima:\n%s", server.body())
	}
}

// TestSMTPTransportKeepsPlainConnectionWhenStartTLSAbsent memastikan server
// tanpa STARTTLS tetap dapat dipakai.
func TestSMTPTransportKeepsPlainConnectionWhenStartTLSAbsent(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		writer := bufio.NewWriter(conn)
		fmt.Fprint(writer, "220 smtp.test ESMTP siap\r\n")
		_ = writer.Flush()
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			command := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(command, "EHLO"):
				fmt.Fprint(writer, "250 smtp.test\r\n")
			case strings.HasPrefix(command, "MAIL"), strings.HasPrefix(command, "RCPT"):
				fmt.Fprint(writer, "250 2.1.0 Ok\r\n")
			case strings.HasPrefix(command, "DATA"):
				fmt.Fprint(writer, "354 lanjut\r\n")
				_ = writer.Flush()
				for {
					dataLine, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if strings.TrimRight(dataLine, "\r\n") == "." {
						break
					}
				}
				fmt.Fprint(writer, "250 2.0.0 Ok\r\n")
			case strings.HasPrefix(command, "QUIT"):
				fmt.Fprint(writer, "221 Bye\r\n")
				_ = writer.Flush()
				return
			default:
				fmt.Fprint(writer, "250 2.0.0 Ok\r\n")
			}
			_ = writer.Flush()
		}
	}()
	defer func() { <-done }()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	sender := NewSMTPSender(SMTPConfig{
		Host:    host,
		Port:    atoi(port),
		From:    "noreply@example.test",
		UseTLS:  false,
		Timeout: 10 * time.Second,
	})
	if err := sender.Send(context.Background(), GenericNotice(
		"pegawai@example.test", "Halo", "Isi", "notif-plain")); err != nil {
		t.Fatalf("pengiriman gagal: %v", err)
	}
}

// atoi mengubah string angka menjadi bilangan bulat.
func atoi(value string) int {
	result := 0
	for _, character := range value {
		if character < '0' || character > '9' {
			continue
		}
		result = result*10 + int(character-'0')
	}
	return result
}
