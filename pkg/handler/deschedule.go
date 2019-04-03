package handler

import "fmt"

type descheduleHandler struct {
}

func (dh descheduleHandler) Handle(event Event) {
	fmt.Println("deschedule event handled")
}
