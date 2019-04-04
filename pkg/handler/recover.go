package handler

import (
	"fmt"
	"time"

	"github.com/lentil1016/descheduler/pkg/predictor"
	"github.com/lentil1016/descheduler/pkg/timer"
)

type recoverHandler struct {
}

func (rh *recoverHandler) Handle(event Event) {
	rs := predictor.GetReplicaSetByKey(event.key)
	if rs != nil {
		if _, ok := recoveringMap[rs.ObjectMeta.Name]; ok {
			delete(recoveringMap, rs.ObjectMeta.Name)
			if len(recoveringMap) == 0 {
				isRecovering = false
				fmt.Println("recoverHandler: ReplicaSets that been evicted have now recovered")
				fmt.Println("Push another schedule event after 5 second...")
				timer.PushTimerEventAfter(5 * time.Second)
			} else {
				fmt.Printf("recoverHandler: Received ReplicaSet %v recover event. Still waiting for %v replica sets recovering\n", rs.ObjectMeta.Name, len(recoveringMap))
			}
		}
	}
}
