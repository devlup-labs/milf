package domain

import "errors"

type QueuePool struct {
	queues map[string]*Queue
}

func NewQueuePool(queues []*Queue) (*QueuePool, error) {
	if len(queues) == 0 {
		return nil, errors.New("queue pool cannot be empty")
	}

	m := make(map[string]*Queue)
	for _, q := range queues {
		if _, ok := m[q.QueueID]; ok {
			return nil, errors.New("duplicate queue ID")
		}
		m[q.QueueID] = q
	}

	return &QueuePool{queues: m}, nil
}

func (p *QueuePool) Get(queueID string) (*Queue, bool) {
	q, ok := p.queues[queueID]
	return q, ok
}

func (p *QueuePool) All() map[string]*Queue {
	return p.queues
}
