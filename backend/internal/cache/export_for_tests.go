package cache

// TestConfig mengembalikan konfigurasi cache untuk pengujian.
func (c *Cache) TestConfig() Config {
	return c.config
}
