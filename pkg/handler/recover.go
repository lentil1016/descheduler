package handler

import "fmt"

type recoverHandler struct {
}

func (rh recoverHandler) Handle(event Event) {
	fmt.Println("recover event handled")
}
