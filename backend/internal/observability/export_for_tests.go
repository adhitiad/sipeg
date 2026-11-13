package observability

import "time"

// SetNow mengatur fungsi waktu untuk pengujian.
func (l *Logger) SetNow(now func() time.Time) {
	l.now = now
}
