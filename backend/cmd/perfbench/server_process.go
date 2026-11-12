package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

// serverProcess adalah proses cmd/server yang dijalankan pengukuran.
type serverProcess struct {
	cmd     *exec.Cmd
	BaseURL string
	log     *os.File
}

// startServer membangun dan menjalankan binari cmd/server terhadap database
// pengukuran.
//
// Server asli dipakai agar rantai handler produksi ikut terukur. Binari
// dibangun dari sumber yang sedang diuji, sehingga perubahan pada kode
// langsung tercermin pada pengukuran.
func startServer(ctx context.Context) (*serverProcess, error) {
	binary, err := buildServer()
	if err != nil {
		return nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	logPath := fmt.Sprintf("%s/simpeg-server-%d.log", os.TempDir(), os.Getpid())
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("buat berkas log server: %w", err)
	}
	cmd := exec.Command(binary)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = serverEnvironment(port)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("jalankan server: %w", err)
	}
	server := &serverProcess{
		cmd:     cmd,
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		log:     logFile,
	}
	if err := server.waitReady(ctx, 60*time.Second); err != nil {
		_ = server.stop()
		return nil, fmt.Errorf("%w (log server: %s)", err, logPath)
	}
	return server, nil
}

// serverEnvironment menyusun environment untuk proses server.
//
// Batas laju dinaikkan supaya pengukuran tidak berhenti pada pembatas laju
// bawaan. Ukuran pool koneksi ditahan di bawah batas pooler sesi Supabase
// supaya galat koneksi tidak mengaburkan pengukuran latensi.
func serverEnvironment(port int) []string {
	environment := append([]string{}, os.Environ()...)
	environment = append(environment,
		"PORT="+strconv.Itoa(port),
		"APP_ENV=development",
		"TRUSTED_ORIGINS=http://127.0.0.1:"+strconv.Itoa(port),
		"RATE_LIMIT_REQUESTS=100000",
		"RATE_LIMIT_AUTH_REQUESTS=100000",
		"DB_MAX_OPEN_CONNS=10",
		"DB_MAX_IDLE_CONNS=10",
	)
	return environment
}

// waitReady menunggu endpoint readiness siap.
func (s *serverProcess) waitReady(ctx context.Context, timeout time.Duration) error {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+"/ready", nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("readiness mengembalikan %d", response.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("server tidak siap: %w", lastErr)
}

// stop menghentikan proses server.
func (s *serverProcess) stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	_ = s.cmd.Process.Kill()
	_ = s.cmd.Wait()
	if s.log != nil {
		return s.log.Close()
	}
	return nil
}

// issuePerfSession membuat sesi admin untuk pengukuran. Token tidak pernah
// dicetak sehingga tidak bocor ke log atau terminal.
//
// Pegawai admin dicari lewat peran, bukan lewat urutan waktu, karena seluruh
// pegawai sintetis ditulis dalam satu batch dan memiliki createdAt yang sama.
func issuePerfSession(ctx context.Context) (string, error) {
	db, err := openDB(ctx)
	if err != nil {
		return "", err
	}
	var admin models.User
	err = db.WithContext(ctx).
		Joins("JOIN \"Role\" ON \"Role\".\"id\" = \"user\".\"roleId\"").
		Where("\"user\".\"aktif\" = ? AND \"Role\".\"name\" = ?", true, "admin").
		First(&admin).Error
	if err != nil {
		return "", fmt.Errorf("cari admin untuk sesi uji: %w", err)
	}
	issuer := auth.NewGormSessionIssuer(db, time.Hour)
	token, _, err := issuer.IssueSession(ctx, admin.ID)
	if err != nil {
		return "", fmt.Errorf("terbitkan sesi uji: %w", err)
	}
	return token, nil
}

// freePort mencari porta TCP yang bebas.
func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("cari porta bebas: %w", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
