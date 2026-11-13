package slo

import "context"

// Exports untuk pengujian pengoleki metrik.
func (c *Collector) TestWindowCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.windows)
}

func (c *Collector) TestFlushAll(ctx context.Context) error {
	return c.flushAll(ctx)
}
