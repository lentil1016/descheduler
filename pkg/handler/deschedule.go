package handler

import (
	"fmt"

	"github.com/lentil1016/descheduler/pkg/predictor"
	"github.com/lentil1016/descheduler/pkg/timer"
)

type descheduleHandler struct {
}

func (dh *descheduleHandler) Handle(event Event) {
	if timer.IsOutOfTime() {
		fmt.Println("Deschedule event aborted by timer")
		return
	}

	// get busy nodes.
	busyNodes, ok := predictor.GetBusyNodes()
	if !ok {
		return
	}
	fmt.Println("descheduleHandler: Deschedule Triggered, start picking Pods")
	pods, err := predictor.GetEvictPods(busyNodes)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("descheduleHandler: Pods picking done, start to evict")
	predictor.Evict(pods)
	recoveringMap = make(map[string]bool, len(pods))
	for _, pod := range pods {
		rsName := predictor.GetPodReplicaSetName(pod)
		if rsName != "" {
			recoveringMap[rsName] = true
		}
	}
	isRecovering = true
	fmt.Println("descheduleHandler: Eviction is finished, waiting for recovering")
}
