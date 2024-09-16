package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// ErrUnknownType is returned when an attempt is made to resolve a value from a provider but the
// requested type is unknown.
var ErrUnknownType = errors.New("requested type is unknown")

// An UnknownType is an [error] indicating that an attempt was made to resolve a value from a
// provider but the requested type was unknown. Calling [errors.Is] with a [UnknownType] and
// [ErrUnknownType] returns true.
type UnknownType struct {

	// Type is the unknown type.
	Type reflect.Type
}

// Error implements [error].
func (err UnknownType) Error() string {
	return fmt.Sprintf("requested type %v is unknown to the provider", err.Type)
}

// Is indicates that a [UnknownType] is [ErrUnknownType].
func (err UnknownType) Is(target error) bool {
	return target == ErrUnknownType
}

// ErrScopedValueRequestedFromRootProvider is returned when an attempt is made to resolve a scoped
// value from a [RootProvider].
var ErrScopedValueRequestedFromRootProvider = errors.New("RootProvider cannot resolve a scoped value")

// A ScopedValueRequestedFromRootProvider is an [error] indicating that an attempt was made to
// resolve a scoped value from a [RootProvider]. Calling [errors.Is] with a
// [ScopedValueRequestedFromRootProvider] and [ErrScopedValueRequestedFromRootProvider] returns
// true.
type ScopedValueRequestedFromRootProvider struct {

	// Type is the unknown type.
	Type reflect.Type
}

// Error implements [error].
func (err ScopedValueRequestedFromRootProvider) Error() string {
	return fmt.Sprintf("RootProvider cannot resolve a scoped value of type %v", err.Type)
}

// Is indicates that a [ScopedValueRequestedFromRootProvider] is [ErrScopedValueRequestedFromRootProvider].
func (err ScopedValueRequestedFromRootProvider) Is(target error) bool {
	return target == ErrScopedValueRequestedFromRootProvider
}

// A RootProvider is a [Provider] that can resolve [Transient] and [Singleton] values.
type RootProvider struct {
	registrations map[reflect.Type]registration
	singletons    *instanceMap
}

func (provider RootProvider) Resolve(typ reflect.Type) (any, error) {
	registration, ok := provider.registrations[typ]
	if !ok {
		return nil, fmt.Errorf("no implementation registered for service type %v", typ)
	}
	switch registration.lifetime {
	case Transient:
		return registration.factory(provider)
	case Scoped:
		return nil, ScopedValueRequestedFromRootProvider{
			Type: typ,
		}
	case Singleton:
		return provider.resolveSingleton(typ, registration.factory)
	default:
		panic("this code should be unreachable: please open a an issue at https://github.com/ttd2089/stahp/issues/new")
	}
}

func (provider RootProvider) resolveSingleton(typ reflect.Type, factory factoryFunc) (any, error) {
	return provider.singletons.resolve(typ, factory, provider)
}

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
