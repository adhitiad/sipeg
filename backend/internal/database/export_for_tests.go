package database

// WithSessionTimeZone mengekspor withSessionTimeZone untuk pengujian.
func WithSessionTimeZone(dsn string) string {
	return withSessionTimeZone(dsn)
}
