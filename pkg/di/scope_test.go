package di

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestScope(t *testing.T) {

	t.Run("Resolve", func(t *testing.T) {

		t.Run("returns UnknownType for unknown type", func(t *testing.T) {
			expectedType := reflect.TypeFor[struct{}]()
			provider, err := Registry{}.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			_, err = scope.Resolve(expectedType)
			if !errors.Is(err, ErrUnknownType) {
				t.Fatalf("expected %q; got %q", ErrUnknownType, err)
			}
			var unknownType UnknownType
			if !errors.As(err, &unknownType) {
				t.Fatalf("expected %v to be %T", err, unknownType)
			}
			if unknownType.Type != expectedType {
				t.Errorf("expected err.Type to be %v; got %v", expectedType, unknownType.Type)
			}
		})

		// distinctCapableStruct is required to observe whether pointers point to the same instance
		// or not because pointers to zero-length structs can be equal even when the pointed values
		// are distinct.
		//
		// From the Golang spec:
		// Pointer types are comparable. Two pointer values are equal if they point to the same
		// variable or if both have value nil. Pointers to distinct zero-size variables may or may
		// not be equal.
		type distinctCapableStruct struct {
			//lint:ignore U1000 Field enabled type to be distinct
			x int
		}

		t.Run("transient instances from the same scope are distinct", func(t *testing.T) {
			registry, err := RegisterType[*distinctCapableStruct, *distinctCapableStruct](Registry{}, Transient)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			a, err := scope.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			b, err := scope.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			if a == b {
				t.Fatalf("instances are the same: %p %p", a, b)
			}
		})

		t.Run("singleton instances from the same scope are the same", func(t *testing.T) {
			registry, err := RegisterType[*distinctCapableStruct, *distinctCapableStruct](Registry{}, Singleton)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			a, err := scope.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			b, err := scope.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			if a != b {
				t.Fatalf("instances are not the same: %p %p", a, b)
			}
		})

		t.Run("scoped instances from the same scope are the same", func(t *testing.T) {
			registry, err := RegisterType[*distinctCapableStruct, *distinctCapableStruct](Registry{}, Scoped)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			a, err := scope.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			b, err := scope.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			if a != b {
				t.Fatalf("instances are not the same: %p %p", a, b)
			}
		})

		t.Run("scoped instances from different scopes are distinct", func(t *testing.T) {
			registry, err := RegisterType[*distinctCapableStruct, *distinctCapableStruct](Registry{}, Scoped)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scopeA := provider.NewScope()
			a, err := scopeA.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			scopeB := provider.NewScope()
			b, err := scopeB.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			if a == b {
				t.Fatalf("instances are the same: %p %p", a, b)
			}
		})

		t.Run("scoped instances from descendant scopes are distinct", func(t *testing.T) {
			registry, err := RegisterType[*distinctCapableStruct, *distinctCapableStruct](Registry{}, Scoped)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scopeA := provider.NewScope()
			a, err := scopeA.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			scopeB := scopeA.NewScope()
			b, err := scopeB.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			if a == b {
				t.Fatalf("instances are the same: %p %p", a, b)
			}
		})
	})

	t.Run("Close", func(t *testing.T) {

		t.Run("closes ContextCloser values", func(t *testing.T) {
			registry, err := RegisterType[*mockContextCloser, *mockContextCloser](Registry{}, Scoped)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			resolved, err := scope.Resolve(reflect.TypeFor[*mockContextCloser]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			closer, ok := resolved.(*mockContextCloser)
			if !ok {
				t.Fatalf("expected Resolve to return %T; got %T", closer, resolved)
			}
			if errs := scope.Close(context.Background()); len(errs) != 0 {
				t.Fatalf("unexpected errors from Close: %v", errs)
			}
			if !closer.closed {
				t.Fatalf("closer was not closed")
			}
		})

		t.Run("returns errors from ContextCloser values", func(t *testing.T) {
			expectedErr := errors.New("expected error")
			registry, err := RegisterFactory[*errorContextCloser](Registry{}, Scoped, func(Resolver) (*errorContextCloser, error) {
				return &errorContextCloser{
					err: expectedErr,
				}, nil
			})
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			_, err = scope.Resolve(reflect.TypeFor[*errorContextCloser]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			errs := scope.Close(context.Background())
			if len(errs) != 1 {
				t.Fatalf("expected 1 error, got %d (%v)", len(errs), errs)
			}
			if !errors.Is(errs[0], expectedErr) {
				t.Fatalf("expected errs[0] to be %v; got %v", expectedErr, errs[0])
			}
		})

		t.Run("closes Closer values", func(t *testing.T) {
			registry, err := RegisterType[*mockCloser, *mockCloser](Registry{}, Scoped)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			resolved, err := scope.Resolve(reflect.TypeFor[*mockCloser]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			closer, ok := resolved.(*mockCloser)
			if !ok {
				t.Fatalf("expected Resolve to return %T; got %T", closer, resolved)
			}
			if errs := scope.Close(context.Background()); len(errs) != 0 {
				t.Fatalf("unexpected errors from Close: %v", errs)
			}
			if !closer.closed {
				t.Fatalf("closer was not closed")
			}
		})

		t.Run("returns errors from Closer values", func(t *testing.T) {
			expectedErr := errors.New("expected error")
			registry, err := RegisterFactory[*errorCloser](Registry{}, Scoped, func(Resolver) (*errorCloser, error) {
				return &errorCloser{
					err: expectedErr,
				}, nil
			})
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			_, err = scope.Resolve(reflect.TypeFor[*errorCloser]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			errs := scope.Close(context.Background())
			if len(errs) != 1 {
				t.Fatalf("expected 1 error, got %d (%v)", len(errs), errs)
			}
			if !errors.Is(errs[0], expectedErr) {
				t.Fatalf("expected errs[0] to be %v; got %v", expectedErr, errs[0])
			}
		})

		t.Run("returns multiple errors from ContextCloser and Closer values", func(t *testing.T) {
			expectedErrs := []error{
				errors.New("first expected error"),
				errors.New("second expected error"),
				errors.New("third expected error"),
			}
			registry, err := RegisterFactory[*errorContextCloser](Registry{}, Scoped, func(Resolver) (*errorContextCloser, error) {
				return &errorContextCloser{
					err: expectedErrs[0],
				}, nil
			})
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			registry, err = RegisterFactory[*errorCloser](registry, Scoped, func(Resolver) (*errorCloser, error) {
				return &errorCloser{
					err: expectedErrs[1],
				}, nil
			})
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			registry, err = RegisterFactory[*errorContextCloser2](registry, Scoped, func(Resolver) (*errorContextCloser2, error) {
				return &errorContextCloser2{
					err: expectedErrs[2],
				}, nil
			})
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			_, err = scope.Resolve(reflect.TypeFor[*errorContextCloser]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			_, err = scope.Resolve(reflect.TypeFor[*errorCloser]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			_, err = scope.Resolve(reflect.TypeFor[*errorContextCloser2]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			errs := scope.Close(context.Background())
			if len(errs) != len(expectedErrs) {
				t.Fatalf("expected %d error, got %d (%v)", len(expectedErrs), len(errs), errs)
			}
			found := 0
			for _, err := range errs {
				for _, expectedErr := range expectedErrs {
					if errors.Is(err, expectedErr) {
						found++
						continue
					}
				}
			}
			if found != len(expectedErrs) {
				t.Fatalf("expected errs to be %v; got %v", expectedErrs, errs)
			}
		})

		t.Run("gives up on blocking calls when context is Done", func(t *testing.T) {
			unexpectedErrs := []error{
				errors.New("first unexpected error"),
				errors.New("second unexpected error"),
			}
			registry, err := RegisterFactory[*blockingContextCloser](Registry{}, Scoped, func(Resolver) (*blockingContextCloser, error) {
				return &blockingContextCloser{
					blockTime: time.Second,
					err:       unexpectedErrs[0],
				}, nil
			})
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			registry, err = RegisterFactory[*blockingCloser](registry, Scoped, func(Resolver) (*blockingCloser, error) {
				return &blockingCloser{
					blockTime: time.Second,
					err:       unexpectedErrs[1],
				}, nil
			})
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			scope := provider.NewScope()
			_, err = scope.Resolve(reflect.TypeFor[*blockingContextCloser]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			_, err = scope.Resolve(reflect.TypeFor[*blockingCloser]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
			defer cancel()
			if errs := scope.Close(ctx); len(errs) != 0 {
				t.Fatalf("unexpected errors from Close: %v", errs)
			}
		})
	})
}

type mockContextCloser struct {
	closed bool
}

func (m *mockContextCloser) Close(context.Context) error {
	m.closed = true
	return nil
}

type mockCloser struct {
	closed bool
}

func (m *mockCloser) Close() error {
	m.closed = true
	return nil
}

type errorContextCloser struct {
	err error
}

func (m *errorContextCloser) Close(context.Context) error {
	return m.err
}

type errorCloser struct {
	err error
}

func (m *errorCloser) Close() error {
	return m.err
}

type errorContextCloser2 errorContextCloser

func (m *errorContextCloser2) Close(context.Context) error {
	return m.err
}

type blockingContextCloser struct {
	blockTime time.Duration
	err       error
}

func (m *blockingContextCloser) Close(context.Context) error {
	<-time.After(m.blockTime)
	return m.err
}

type blockingCloser struct {
	blockTime time.Duration
	err       error
}

func (m *blockingCloser) Close(context.Context) error {
	<-time.After(m.blockTime)
	return m.err
}
