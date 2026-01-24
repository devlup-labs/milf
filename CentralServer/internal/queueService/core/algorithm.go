package core

import(
	"central_server/internal/queueService/domain"
)

type QueueSelector struct{}

func (q *QueueSelector)SelectQueue(pool domain.QueuePool, maxRam int) *domain.Queue {
	var aptKey string
	for key,queue := range pool.All() {
		if maxRam > queue.ResourceRange.MaxRam{
			aptKey = key
		}else if(maxRam < queue.ResourceRange.MinRam){
			break
		}
	}
	
	return pool.All()[aptKey]
}