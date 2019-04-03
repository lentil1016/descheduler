package handler

import (
	"fmt"

	"github.com/lentil1016/descheduler/pkg/predictor"
	"github.com/lentil1016/descheduler/pkg/timer"
)

type descheduleHandler struct {
}

func (dh *descheduleHandler) Handle(event Event) {
	if !isTriggered() {
		return
	}
	fmt.Println("descheduleHandler: Deschedule Triggered, start picking Pods.")

	isRecovering = true
	fmt.Println("deschedule event handled")
}

func isTriggered() bool {
	if timer.IsOutOfTime() {
		fmt.Println("Deschedule event droped by timer.")
		return false
	}
	// check if nodes are in low spared or in high spared.
	if _, _, ok := predictor.InBadSparedState(); ok {
		return true
	} else {
		return false
	}
}
