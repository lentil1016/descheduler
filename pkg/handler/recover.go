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
				fmt.Println("recover event handled")
			} else {
				fmt.Printf("Received replica set %v recover event. Still waiting for %v replica sets recovering\n", rs.ObjectMeta.Name, len(recoveringMap))
			}
		}
	}
}
