package trigger

import (
	"github.com/lentil1016/descheduler/pkg/timer"
	"k8s.io/client-go/tools/cache"
)

func IsTriggered(nodIndexer cache.Indexer) bool {
	if timer.IsOutOfTime() {
		return false
	}
	return true
}
