package handler

type Event struct {
	key          string
	eventType    string
	resourceType string
}

type eventHandler interface {
	Handle(event Event)
}

// There is no race condition on this value because there is only one worker thread
// So only one event will be handled at a time
var isRecovering = false
var recoveringMap map[string]bool

func NewEvent(key, eventType, resourceType string) Event {
	return Event{
		key:          key,
		eventType:    eventType,
		resourceType: resourceType,
	}
}

func Type(event Event) eventHandler {
	if isRecovering {
		// Handle recover event when the replica sets is recovering
		if event.resourceType == "replicaSet" {
			return &recoverHandler{}
		}
	} else {
		// Handle deschedule event only when replicas number of the replica sets
		// that is being descheduled last time have recovered.
		if event.resourceType == "timer" || event.resourceType == "node" {
			return &descheduleHandler{}
		}
	}
	return defaultHandler{}
}

type defaultHandler struct{}

func (dh defaultHandler) Handle(event Event) {
}
