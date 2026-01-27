package core
import (
	"central_server/internal/queueService/domain"
	"central_server/utils"
)

type QueueSelector struct{}

func (q *QueueSelector)SelectQueue(pool domain.QueuePool, maxRam int) *domain.Queue {
	var aptKey string
	firstKey := ""
	for key, queue := range pool.All() {
		if firstKey == "" {
			firstKey = key
		}
		if maxRam > queue.ResourceRange.MaxRam {
			aptKey = key
		} else if maxRam < queue.ResourceRange.MinRam {
			break
		}
	}
	if aptKey == "" {
		aptKey = firstKey
	}
	utils.Info("Selected queue: " + aptKey)
	return pool.All()[aptKey]
}