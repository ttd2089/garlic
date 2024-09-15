package di

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
)

// ErrNonConcreteImplementation is returned when an attempt is made to register an implementation
// that is not a concrete type.
var ErrNonConcreteImplementation = errors.New("implementation type must be concrete")

// A NonConcreteImplementation is an [error] indicating that an attempt was made to register an
// implementation type that is not a concrete type. Calling [errors.Is] with a
// NonConcreteImplementation and [ErrNonConcreteImplementation] returns true.
type NonConcreteImplementation struct {

	// Type is the non-concrete type.
	Type reflect.Type
}

// Error implements [error].
func (err NonConcreteImplementation) Error() string {
	return fmt.Sprintf("implementation type %v is not a concrete type", err.Type)
}

// Is indicates that a [NonConcreteImplementation] is [ErrNonConcreteImplementation].
func (err NonConcreteImplementation) Is(target error) bool {
	return target == ErrNonConcreteImplementation
}

// ErrInvalidImplementation is returned when an attempt is made to register an implementation type
// for a target type is not assignable to.
var ErrInvalidImplementation = errors.New("implementation type is not assignable to target type")

// An InvalidImplementation is an [error] indicating that an attempt was made to register an
// implementation type for a target type it is not assignable to. Calling [errors.Is] with an
// [InvalidImplementation] and [ErrInvalidImplementation] returns true.
type InvalidImplementation struct {

	// Type is the type that cannot be assigned to [InvalidImplementation.Target].
	Type reflect.Type

	// Target is the type to which [InvalidImplementation.Type] cannot be assigned.
	Target reflect.Type
}

// Error implements [error].
func (err InvalidImplementation) Error() string {
	return fmt.Sprintf(
		"implementation type %v is not assignable to target type %v",
		err.Type,
		err.Target)
}

// Is indicates that an [InvalidImplementation] is [ErrInvalidImplementation].
func (err InvalidImplementation) Is(target error) bool {
	return target == ErrInvalidImplementation
}

// ErrUndefinedLifetime is returned when an attempt is made to register a type with a [Lifetime]
// whose value is not one of the defined values [Transient], [Scoped], or [Singleton].
var ErrUndefinedLifetime = errors.New("undefined lifetime")

// An UndefinedLifetime is an [error] indicating that an attempt was made to register a type with a
// [Lifetime] whose value is not one of the defined values [Transient], [Scoped], or [Singleton].
// Calling [errors.Is] with an [UndefinedLifetime] and [ErrUndefinedLifetime] returns true.
type UndefinedLifetime struct {

	// Value is the undefined value.
	Value Lifetime
}

// Error implements [error].
func (err UndefinedLifetime) Error() string {
	return fmt.Sprintf("undefined lifetime: %d", int(err.Value))
}

// Is indicates that an [UndefinedLifetime] is [ErrUndefinedLifetime].
func (err UndefinedLifetime) Is(target error) bool {
	return target == ErrUndefinedLifetime
}

// ErrUnsharableType is returned when an unsharable type is registered with a [Lifetime] other than
// [Transient].
var ErrUnsharableType = errors.New("unsharable type cannot be registered with non-Transient lifetime")

// A UnsharableType is an [error] indicating that an attempt was made to register an unsharable
// type with a [Lifetime] other than [Transient]. Calling [errors.Is] with a [UnsharableType] and
// [ErrUnsharableType] returns true.
type UnsharableType struct {

	// Type is the unsharable type.
	Type reflect.Type

	// Lifetime is the non-transient lifetime.
	Lifetime Lifetime
}

// Error implements [error].
func (err UnsharableType) Error() string {
	return fmt.Sprintf(
		"unsharable type %v cannot be registered with non-Transient Lifetime %v",
		err.Type,
		err.Lifetime)
}

// Is indicates that a [UnsharableType] is [ErrUnsharableType].
func (err UnsharableType) Is(target error) bool {
	return target == ErrUnsharableType
}

// ErrNoDefaultFactory is returned when an attempt is made to register an implementation type for
// which the package cannot provide a default factory to obtain instances from.
var ErrNoDefaultFactory = errors.New("implementation type has no default factory")

// An NoDefaultFactory is an [error] indicating that an attempt was made to register an
// implementation type for which the package cannot provide a default factory to obtain instances
// from. Calling [errors.Is] with a [NoDefaultFactory] and [ErrNoDefaultFactory] returns true.
type NoDefaultFactory struct {

	// Type is the type for which the package cannot provide a default factory.
	Type reflect.Type
}

// Error implements [error].
func (err NoDefaultFactory) Error() string {
	return fmt.Sprintf("implementation type %v has no default factory", err.Type)
}

// Is indicates that an [NoDefaultFactory] is [ErrNoDefaultFactory].
func (NoDefaultFactory) Is(target error) bool {
	return target == ErrNoDefaultFactory
}

// ErrNilFactory is returned when an attempt is made to register a nil factory.
var ErrNilFactory = errors.New("factory cannot be nil")

// A Registry is a collection into which services can be registered and from which a
// [RootProvider] may be built.
type Registry struct {
	registrations map[reflect.Type]registration
}

func (r Registry) BuildRootProvider() (RootProvider, error) {
	return RootProvider{
		registrations: maps.Clone(r.registrations),
		singletons:    &instanceMap{},
	}, nil
}

// RegisterType is a shorthand for calling [RegisterFactory] using the result of calling
// [GetDefaultFactory] for the [Impl] type.
func RegisterType[Target any, Impl any](registry Registry, lifetime Lifetime) (Registry, error) {
	factory, err := GetDefaultFactory[Impl]()
	if err != nil {
		return registry, err
	}
	return RegisterFactory[Target](registry, lifetime, factory)
}

// A Factory is a function that makes instances of T using a Resolver to initialize dependencies.
type Factory[T any] func(Resolver) (T, error)

func RegisterFactory[Target any, Impl any](
	registry Registry,
	lifetime Lifetime,
	factory Factory[Impl],
) (Registry, error) {

	target := reflect.TypeFor[Target]()
	impl := reflect.TypeFor[Impl]()

	if err := validateRegistrationTypes(target, impl); err != nil {
		return registry, err
	}

	if err := validateLifetime(impl, lifetime); err != nil {
		return registry, err
	}

	if factory == nil {
		return registry, ErrNilFactory
	}

	return addRegistration(registry, target, registration{
		lifetime: lifetime,
		factory: func(resolver Resolver) (any, error) {
			return factory(resolver)
		},
	}), nil
}

func validateRegistrationTypes(target reflect.Type, impl reflect.Type) error {

	if !isConcrete(impl) {
		return NonConcreteImplementation{
			Type: impl,
		}
	}

	if !impl.AssignableTo(target) {
		return InvalidImplementation{
			Target: target,
			Type:   impl,
		}
	}

	return nil
}

func validateLifetime(impl reflect.Type, lifetime Lifetime) error {

	if _, ok := knownLifetimes[lifetime]; !ok {
		return UndefinedLifetime{
			Value: lifetime,
		}
	}

	if lifetime != Transient && !isSharable(impl) {
		return UnsharableType{
			Type:     impl,
			Lifetime: lifetime,
		}
	}

	return nil
}

func isConcrete(typ reflect.Type) bool {
	return typ.Kind() != reflect.Interface
}

func isSharable(typ reflect.Type) bool {
	kind := typ.Kind()
	if kind == reflect.Pointer {
		return true
	}
	if kind == reflect.Chan {
		return true
	}
	return false
}

type factoryFunc func(Resolver) (any, error)

type registration struct {
	lifetime Lifetime
	factory  factoryFunc
}

func addRegistration(registry Registry, target reflect.Type, registration_ registration) Registry {
	if registry.registrations == nil {
		registry.registrations = make(map[reflect.Type]registration, 0)
	}
	registry.registrations[target] = registration_
	return registry
}
