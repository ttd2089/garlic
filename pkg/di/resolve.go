package di

import (
	"errors"
	"fmt"
	"reflect"
)

// ErrNilResolver is returned when the [Resolve] function receives a nil [Resolver] argument.
var ErrNilResolver = errors.New("cannot resolve instances from nil Resolver")

// ErrResolverError is returned when the [Resolve] function receives an error from a [Resolver].
var ErrResolverError = errors.New("Resolver returned error")

type resolverError struct {
	wrapped error
}

// Error implements [error].
func (err resolverError) Error() string {
	return fmt.Sprintf("resolver error: %v", err.wrapped)
}

// Is indicates that a [resolverError] is [ErrResolverError].
func (resolverError) Is(target error) bool {
	return target == ErrResolverError
}

// Unwrap gets the underlying [error] from the [Resolver].
func (err resolverError) Unwrap() error {
	return err.wrapped
}

// ErrInvalidResolution is returned when the [Resolve] function receives a value from a [Resolver]
// that cannot be assigned to the requested type.
//
// NOTE: This error ALWAYS points to a broken implementation of [Resolver]. The contract of
// [Resolver] requires returned values to be assignable to the requested type, but the absence of
// generic methods in Go's type system prevents this from being enforced statically.
var ErrInvalidResolution = errors.New("value from Resolver does not have requested type")

// A InvalidResolution is an [error] indicating that a [Resolver] returned a a value that could not
// be assigned to the requested type. Calling [errors.Is] with an [InvalidResolution] and
// [ErrInvalidResolution] returns true.
type InvalidResolution struct {

	// Requested is the type that was requested from the [Resolver].
	Requested reflect.Type

	// Returned is the type of the value the [Resolver] returned.
	Returned reflect.Type
}

// Error implements [error].
func (err InvalidResolution) Error() string {
	return fmt.Sprintf(
		"value from Resolver has type %v when %v was requested",
		err.Returned,
		err.Requested)
}

// Is indicates that an [InvalidResolution] is [ErrInvalidResolution].
func (InvalidResolution) Is(target error) bool {
	return target == ErrInvalidResolution
}

// A Resolver resolves instances of a requested type.
type Resolver interface {

	// Resolve provides an instance of the requested type if one is registered. Implementations
	// MUST ensure that the values returned are assignable to the requested type.
	Resolve(reflect.Type) (any, error)
}

// Resolve obtains an instance of the requested type from a [Resolver]. An [error] is returned when
// the [Resolver] returns an [error] or a value that is not assignable to T.
func Resolve[T any](resolver Resolver) (T, error) {
	if resolver == nil {
		var zero T
		return zero, ErrNilResolver
	}

	var zero T
	typ := reflect.TypeFor[T]()

	resolved, err := resolver.Resolve(typ)
	if err != nil {
		return zero, resolverError{wrapped: err}
	}

	typed, ok := resolved.(T)
	if !ok {
		return zero, InvalidResolution{
			Requested: typ,
			Returned:  reflect.TypeOf(resolved),
		}
	}

	return typed, nil
}
