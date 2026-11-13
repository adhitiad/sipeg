package outbox

import "time"

// TestBackoff mengekspor perhitungan backoff untuk pengujian.
func (d *Dispatcher) TestBackoff(attempt int) time.Duration {
	return d.backoff(attempt)
}
