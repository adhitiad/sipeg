package qstash

// Exports untuk pengujian publisher.
func (p *Publisher) TestPublishEndpoint() string {
	return p.publishEndpoint()
}

func (p *Publisher) TestSchedulesURL() string {
	return p.schedulesURL()
}
