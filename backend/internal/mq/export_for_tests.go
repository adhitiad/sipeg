package mq

import "context"

// Encode/DecodeKind diekspor untuk pengujian.
func Encode(kind string) ([]byte, error) {
	return encode(kind)
}

func DecodeKind(body []byte) (string, error) {
	return decodeKind(body)
}

// TestHandlerCount mengembalikan jumlah handler terdaftar.
func (c *Consumer) TestHandlerCount() int {
	return len(c.handlers)
}

// TestHandler mengembalikan handler terdaftar untuk pengujian.
func (c *Consumer) TestHandler(kind string) (JobFunc, bool) {
	job, ok := c.handlers[kind]
	return job, ok
}

var _ = context.Background
