// Package observability menyediakan pencatatan terstruktur dan korelasi
// permintaan untuk backend SIMPEG.
package observability

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// Fields adalah pasangan key-value tambahan pada satu entri log.
//
// Fields berupa alias dari map[string]any agar logger pada paket ini dapat
// dipakai sebagai port pencatatan oleh paket lain.
type Fields = map[string]any

// Logger menulis entri log berformat JSON satu baris per entri.
type Logger struct {
	mu  sync.Mutex
	out io.Writer
	now func() time.Time
}

// New membuat logger yang menulis ke out. out nil berarti log dibuang.
func New(out io.Writer) *Logger {
	return &Logger{out: out, now: time.Now}
}

// Info menulis entri log level info.
func (l *Logger) Info(message string, fields Fields) { l.write("info", message, fields) }

// Warn menulis entri log level warn.
func (l *Logger) Warn(message string, fields Fields) { l.write("warn", message, fields) }

// Error menulis entri log level error.
func (l *Logger) Error(message string, fields Fields) { l.write("error", message, fields) }

// Now mengembalikan waktu saat ini; aman dipanggil pada logger nil.
func (l *Logger) Now() time.Time {
	if l == nil || l.now == nil {
		return time.Now()
	}
	return l.now()
}

func (l *Logger) write(level, message string, fields Fields) {
	if l == nil || l.out == nil {
		return
	}
	entry := make(map[string]any, len(fields)+3)
	for key, value := range fields {
		entry[key] = value
	}
	entry["ts"] = l.now().UTC().Format(time.RFC3339Nano)
	entry["level"] = level
	entry["msg"] = message

	encoded, err := json.Marshal(entry)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.out.Write(append(encoded, '\n'))
}
