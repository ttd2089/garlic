# garlic

Golang Automated Resource Locator and Injection Container

## Overview

The [`di`][di] package provides mechanisms for declarative, request scope capable dependency injection.

Most applications using [`di`][di] will look something like this:

- Create a [`di.Registry`][di.Registry]
- Register implementations for required types
- Create a [`di.RootProvider`][di.RootProvider] from the [`di.Registry`][di.Registry]
- Resolve the values required to run the application from the [`di.RootProvider`][di.RootProvider]
- Run the application

Many applications can benefit from using request-scoped values. In that case the process will have a few additional steps:

- Create a [`di.Registry`][di.Registry]
- Register implementations for required types
- Create a [`di.RootProvider`][di.RootProvider] from the [`di.Registry`][di.Registry]
- Resolve the values start accepting requests from the [`di.RootProvider`][di.RootProvider]
- Start accepting requests
- For each request:
  - Create a [`di.Scope`][di.Scope] from the [`di.RootProvider`][di.RootProvider]
  - Resolve the values required to handle the request from the [`di.Scope`][di.Scope]
  - Handle the request

## Concepts

### Registries

A [`di.Registry`][di.Registry] is essentially a builder for a [`di.RootProvider`][di.RootProvider]. After creating a [`di.Registry`][di.Registry] you'll add [registrations](#registrations) then build a [`di.RootProvider`][di.RootProvider] that uses those [registrations](#registrations) to resolve and provide values.

### Registrations

A registration is mapping from a [target type](#target-types) to [factory](#factories) returning an instances of an [implementation type](#implementation-types) that implements the [target type](#target-types). The [factory] describes how to obtain a value when the [target type](#target-types) is requested from a [resolver](#resolvers). The registration also includes a [lifetime](#lifetimes) which describes the [resolver](#resolvers) should initialize new values and when it should reuse values it has already initialized and returned for previous requests.

### Target Types

A target type is the type that a [registration](#registrations) describes how to resolve.

### Implementation Types

An implementation type is the concrete type of the value that will be resolved when a target type is requested. Implementation types MUST implement their corresponding [target type](#target-types) and MUST be concrete types. Interface types cannot be implementation types.

### Resolvers

A [`di.Resolver`][di.Resolver] is a value that resolves instances of various types on demand at runtime.

The [`di`][di] package provides two [`di.Resolver`][di.Resolver] implementations: [`di.RootProvider`][di.RootProvider] and [`di.Scope`][di.Scope].

### Factories

A [`di.Factory`][di.Factory] is a function that creates values and initializes their dependencies using a [`di.Resolver`][di.Resolver].

The [`di`][di] package can provide [default factories](#default-factories) for many types, in addition to supporting custom [`di.Factory`][di.Factory] implementations.

### Default Factories

The [`di`][di] package is able to create and initialize many types without requiring users to provide an explicit [factory](#factories).

The default factory for any struct type starts with the zero value for the type, then initializes all of the exported members using the [`di.Resolver`][di.Resolver]. _NOTE_ that the exported members are initialized with whichever factory the [`di.Resolver`][di.Resolver] has registered for its type which is not necessarily a default factory.

The default factory for `bool`, numeric, array, and string types provide the zero value. This includes any type whose [`reflect.Kind`][reflect.Kind] is `reflect.Bool`, `reflect.Int`, `reflect.Int8`, `reflect.Int16`, `reflect.Int32`, `reflect.Int64`, `reflect.Uint`, `reflect.Uint8`, `reflect.Uint16`, `reflect.Uint32`, `reflect.Uint64`, `reflect.Float32`, `reflect.Float64`, `reflect.Complex64`, `reflect.Complex128`, `reflect.Array`, or `reflect.String`.

The default factory for channels provides an unbuffered channel.

The default factory for maps is equivalent to `make(map[T]U)`. This ensures that the returned value can be written to.

The default factory for slices returns `nil`. Unlike with maps, a `nil` slice can be appended to. A zero-length slice can only be written to with an append so semantically there's no difference between `nil` and an empty slice.

Default factories are unavailable for types whose direct [`reflect.Kind`][reflect.Kind] is [`reflect.Uintptr`], [`reflect.Func`], or [`reflect.UnsafePointer`].

Default factories are unavailable for types whose direct [`reflect.Kind`][reflect.Kind] is [`reflect.Interface`], but this is irrelevant because interfaces cannot be [implementation types](#implementation-types).

For types whose [`reflect.Kind`][reflect.Kind] is [`reflect.Pointer`], a default factory is available if and only if there is a default factory for the pointed-to type. For example there is a default factory for `*struct{ X int; Y int }` because there is a default factory for `{ X int; Y int }`, and there is no default facotory for `*func()` because here is no default factory for `func()`. This applies recursively; i.e. there is a default factory for `**struct{ X int; Y int }` because there is a default factory for `*{ X int; Y int }`, and there is no default facotory for `**func()` because here is no default factory for `*func()`. Default factories for pointers initialize the pointed-to value using the corresponding type's default factory, and a pointer to that value. Again, this is recursive; i.e. the default factory for `**struct{ X int; Y int}` initializes a `*struct{ X int; Y int }` and a non-nil pointer to it. The `*struct{ X int; Y int }` is also a pointer type so its default factory initializes a `struct{ X int; Y int}` and a non-nil pointer to it.

### Lifetimes

A [`di.Lifetime`][di.Lifetime] describes when a [`di.Resolver`][di.Resolver] should initialize a new instance of a value to return and when it should reuse a value it has already returned.

The [`di.Transient`][di.Transient] [lifetime][di.Lifetime] specifies that a new value should be initialized every time a type is resolved and can be used with any type.

The [`di.Scoped`][di.Scoped] [lifetime][di.Lifetime] specifies that a single instance of the registered type should be reused every time the type is resolved from the same [`di.Scope`][di.Scope], and the [`di.Singleton`][di.Singleton] [lifetime][di.Lifetime] specifies that a single instance should be reused every time the type is resolved from the same [`di.RootProvider`][di.RootProvider] or any [`di.Scope`][di.Scope] created from it. In order to support reusing the same instance the [`di.Scoped`][di.Scoped] and [`di.Singleton`][di.Singleton] [lifetimes][di.Lifetime] can only be used with [sharable types](#sharable-types).

### Sharable Types

It's not actually possible in Go to return the same instance of a value more than once; we can only return a copy. However, for types' whose values are references to the data we're interested a copy will generally point to the same data. In this case we can return distinct values that each reference the data we want to share. We refer to these as "sharable types". _NOTE_ that since it is the _value_ rather than the identifier that refers to the shared value, assigning a new value to a field that currently holds reference to a shared value will not update the shared value.

All pointer types are sharable because their value is just a reference to an instance of their element type. _NOTE_ that pointer types don't provide any mechanisms to make concurrent use safe, that's up to the type being pointed to. Keep this in mind when writing and pointers for shared values.

Channels are not technically pointers, but copies of channels are readers and writers of the same stream of data and are safe for concurrent use so channels are considerable sharable.

#### Unsharable Types

Arrays are not sharable because the value of the array includes all of its element values. A copy of an array of a copy of each element  After a copy, mutations to one array are not reflected in the other.

Slices hold their element data in underlying arrays by reference, but the portion of the underlying array that the slice exposes and the underlying array itself can change when the slice is modified. As a result, copies of slices will not stay in sync as they are used and are therefore not considered sharable.

Maps are currently _not_ considered sharable. Although maps hold heir data in an underlying structure and copies of maps are consistently observed to reflect writes across instances, the Golang spec does not seem to guarantee that a map write _won't_ result in the written-to map allocating new underlying storage and diverging from the instances with which is was previously consistent (the same way slices can).

[di]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di
[di.Factory]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Factory
[di.Lifetime]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Lifetime
[di.Registry]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Registry
[di.Resolver]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Resolver
[di.RootProvider]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Provider
[di.Singleton]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Singleton
[di.Scope]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Scope
[di.Scoped]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Scoped
[di.Transient]: https://pkg.go.dev/github.com/ttd2089/garlic/pkg/di#Transient

[reflect.Kind]: https://pkg.go.dev/reflect#Kind
