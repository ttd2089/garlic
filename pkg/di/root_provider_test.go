package di

import (
	"errors"
	"reflect"
	"testing"
)

func TestRootProvider(t *testing.T) {

	t.Run("Resolve", func(t *testing.T) {

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

		t.Run("scoped instances cannot be resolved from RootProvider", func(t *testing.T) {
			registry, err := RegisterType[*distinctCapableStruct, *distinctCapableStruct](Registry{}, Scoped)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			_, err = provider.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if !errors.Is(err, ErrScopedValueRequestedFromRootProvider) {
				t.Fatalf("expected err=%v; got %v", ErrScopedValueRequestedFromRootProvider, err)
			}
		})

		t.Run("transient instances from the same provider are distinct", func(t *testing.T) {
			registry, err := RegisterType[*distinctCapableStruct, *distinctCapableStruct](Registry{}, Transient)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			a, err := provider.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			b, err := provider.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			if a == b {
				t.Fatalf("instances are the same: %p %p", a, b)
			}
		})

		t.Run("singleton instances from the same provider are the same", func(t *testing.T) {
			registry, err := RegisterType[*distinctCapableStruct, *distinctCapableStruct](Registry{}, Singleton)
			if err != nil {
				t.Fatalf("unexpected error from RegisterType: %v", err)
			}
			provider, err := registry.BuildRootProvider()
			if err != nil {
				t.Fatalf("unexpected error from BuildRootProvider: %v", err)
			}
			a, err := provider.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			b, err := provider.Resolve(reflect.TypeFor[*distinctCapableStruct]())
			if err != nil {
				t.Fatalf("unexpected error from Resolve: %v", err)
			}
			if a != b {
				t.Fatalf("instances are not the same: %p %p", a, b)
			}
		})
	})
}
