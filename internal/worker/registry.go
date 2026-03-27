package worker

import "sync"

type Registry struct {
	mu            sync.RWMutex
	handlers      map[string]Handler
	defaultHandle Handler
}

func NewRegistry(defaultHandler Handler) *Registry {
	if defaultHandler == nil {
		defaultHandler = NopHandler{}
	}
	return &Registry{
		handlers:      make(map[string]Handler),
		defaultHandle: defaultHandler,
	}
}

func (r *Registry) Register(queueName string, handler Handler) {
	if handler == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[queueName] = handler
}

func (r *Registry) Resolve(queueName string) Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if handler, ok := r.handlers[queueName]; ok {
		return handler
	}
	return r.defaultHandle
}
