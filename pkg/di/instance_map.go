package di

import (
	"reflect"
	"sync"
)

type instanceMap struct {
	mu        sync.RWMutex
	instances map[reflect.Type]any
}

func (m *instanceMap) resolve(
	typ reflect.Type,
	factory factoryFunc,
	resolver Resolver,
) (any, error) {
	if v, ok := m.get(typ); ok {
		return v, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// We may have resolved and saved a singleton instance while we were waiting for a lock so check again.
	if service, ok := m.instances[typ]; ok {
		return service, nil
	}
	// Build, save, and return the scoped instance.
	service, err := factory(resolver)
	if err != nil {
		return nil, err
	}
	if m.instances == nil {
		m.instances = make(map[reflect.Type]any)
	}
	m.instances[typ] = service
	return service, nil
}

func (m *instanceMap) get(typ reflect.Type) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.instances[typ]
	return v, ok
}

func (m *instanceMap) values() []any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]any, 0, len(m.instances))
	for _, v := range m.instances {
		values = append(values, v)
	}
	return values
}
