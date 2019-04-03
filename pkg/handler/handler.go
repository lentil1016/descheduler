package handler

type Event struct {
	key          string
	eventType    string
	resourceType string
}

type eventHandler interface {
	Handle(event Event)
}

func NewEvent(key, eventType, resourceType string) Event {
	return Event{
		key:          key,
		eventType:    eventType,
		resourceType: resourceType,
	}
}

func Type(event Event) eventHandler {
	if event.resourceType == "timer" || event.resourceType == "node" {
		return descheduleHandler{}
	} else if event.resourceType == "replicaSet" {
		return recoverHandler{}
	}
	return defaultHandler{}
}

type defaultHandler struct{}

func (dh defaultHandler) Handle(event Event) {
}
