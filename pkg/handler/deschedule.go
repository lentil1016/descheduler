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
	evictSize := 2
	pods, err := predictor.GetEvictPods(busyNodes, evictSize)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("descheduleHandler: Pods picking done, start to evict")
	predictor.Evict(pods)
	isRecovering = true
	fmt.Println("deschedule event handled")
}
