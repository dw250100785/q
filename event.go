package q

// usage inside q.go: Events field & builder struct.

type (
	// Events the type for registered listeners, it's just a map[string][]func(...interface{})
	Events map[string]EventListeners
	// EventListeners the listeners type, it's just a []func(...interface{})
	EventListeners []func(...interface{})

	// EventEmmiter is the message/or/event manager
	EventEmmiter interface {
		// On registers a particular listener for an event, func receiver parameter(s) is/are optional
		On(string, func(...interface{}))
		// Emit fires a particular event, second parameter(s) is/are optional, if filled then the event has some information to send to the listener
		Emit(string, ...interface{})
		// RemoveEvent receives an event and destroys/ de-activates/unregisters the listeners belongs to it( the event)
		// returns true if something has been removed, otherwise false
		RemoveEvent(string) bool
	}

	eventEmmiter struct {
		evtListeners Events
	}
)

func (e Events) copyTo(emmiter EventEmmiter) {
	if e != nil && len(e) > 0 {
		// register the events to/with their listeners
		for evt, listeners := range e {
			if len(listeners) > 0 {
				for i := range listeners {
					emmiter.On(evt, listeners[i])
				}
			}
		}
	}
}

var _ EventEmmiter = &eventEmmiter{}

func (e *eventEmmiter) On(evt string, listener func(data ...interface{})) {
	if e.evtListeners == nil {
		e.evtListeners = Events{}
	}
	if e.evtListeners[evt] == nil {
		e.evtListeners[evt] = EventListeners{}
	}
	e.evtListeners[evt] = append(e.evtListeners[evt], listener)
}

func (e *eventEmmiter) Emit(evt string, data ...interface{}) {
	if e.evtListeners == nil {
		return // has no listeners to emit/speak yet
	}
	if listeners := e.evtListeners[evt]; listeners != nil && len(listeners) > 0 { // len() should be just fine, but for any case on future...
		for i := range listeners {
			l := listeners[i]
			l(data...)
		}
	}
}

func (e *eventEmmiter) RemoveEvent(evt string) bool {
	if e.evtListeners == nil {
		return false // has nothing to remove
	}

	if listeners := e.evtListeners[evt]; listeners != nil && len(listeners) > 0 { // len() should be just fine, but for any case on future...
		e.evtListeners[evt] = EventListeners{} // some memory allocations here, no worries
		return true
	}

	return false
}
