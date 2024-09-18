package di

import (
	"errors"
	"fmt"
	"reflect"
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

// NewScope creates a new [Scope] which can resolve [Scoped] values as well as [Transient]
// and [Singleton] values.
func (provider RootProvider) NewScope() Scope {
	return Scope{
		root:         provider,
		scopedValues: &instanceMap{},
	}
}

// Resolve returns an instance of the requested type if it was registered as a Transient or
// Singleton value.
func (provider RootProvider) Resolve(typ reflect.Type) (any, error) {
	registration, ok := provider.registrations[typ]
	if !ok {
		return nil, UnknownType{
			Type: typ,
		}
	}
	switch registration.lifetime {
	case Transient:
		return registration.factory(provider)
	case Scoped:
		return nil, ScopedValueRequestedFromRootProvider{
			Type: typ,
		}
	case Singleton:
		return provider.singletons.resolve(typ, registration.factory, provider)
	default:
		panic("this code should be unreachable: please open a an issue at https://github.com/ttd2089/stahp/issues/new")
	}
}
