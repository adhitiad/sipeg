package config

import "time"

// Ekspor konstanta bawaan untuk pengujian di backend/tests/config.
const (
	DefaultDBMaxOpenConns     = defaultDBMaxOpenConns
	DefaultDBMaxIdleConns     = defaultDBMaxIdleConns
	DefaultDBConnMaxLifetime  = defaultDBConnMaxLifetime
	DefaultDBStatementTimeout = defaultDBStatementTimeout
	DefaultSessionLifetime    = defaultSessionLifetime
)

var _ = time.Second
