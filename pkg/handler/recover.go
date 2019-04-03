package handler

import "fmt"

type recoverHandler struct {
}

func (rh *recoverHandler) Handle(event Event) {
	isRecovering = false
	fmt.Println("recover event handled")
}
