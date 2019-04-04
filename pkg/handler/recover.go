package handler

import (
	"fmt"

	"github.com/lentil1016/descheduler/pkg/predictor"
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
			} else {
				fmt.Printf("recoverHandler: Received ReplicaSet %v recover event. Still waiting for %v replica sets recovering\n", rs.ObjectMeta.Name, len(recoveringMap))
			}
		}
	}
}
