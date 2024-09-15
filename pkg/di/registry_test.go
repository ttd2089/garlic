package di

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
	"unsafe"
)

func TestRegistry(t *testing.T) {

	t.Run("RegisterType", func(t *testing.T) {

		t.Run("returns NonConcreteImplementation when Impl is an interface", func(t *testing.T) {
			_, err := RegisterType[io.Reader, io.ReadWriter](Registry{}, Transient)
			if !errors.Is(err, ErrNonConcreteImplementation) {
				t.Fatalf("expected %q; got %q", ErrNonConcreteImplementation, err)
			}
			var nonConcreteImpl NonConcreteImplementation
			if !errors.As(err, &nonConcreteImpl) {
				t.Fatalf("expected %v to be %T", err, nonConcreteImpl)
			}
			if type_ := reflect.TypeFor[io.ReadWriter](); nonConcreteImpl.Type != type_ {
				t.Errorf("expected err.Type to be %v; got %v", type_, nonConcreteImpl.Type)
			}
		})

		t.Run("returns InvalidImplementation when Impl cannot be assigned to Target", func(t *testing.T) {
			_, err := RegisterType[string, struct{}](Registry{}, Transient)
			if !errors.Is(err, ErrInvalidImplementation) {
				t.Fatalf("expected %q; got %q", ErrInvalidImplementation, err)
			}
			var invalidImpl InvalidImplementation
			if !errors.As(err, &invalidImpl) {
				t.Fatalf("expected %v to be %T", err, invalidImpl)
			}
			if strType := reflect.TypeFor[string](); invalidImpl.Target != strType {
				t.Errorf("expected err.Target to be %v; got %v", strType, invalidImpl.Target)
			}
			if structType := reflect.TypeFor[struct{}](); invalidImpl.Type != structType {
				t.Errorf("expected err.Impl to be %v; got %v", structType, invalidImpl.Type)
			}
		})

		t.Run("no default factory", func(t *testing.T) {

			testCases := []struct {
				name        string
				fn          func() (Registry, error)
				expectedErr NoDefaultFactory
			}{
				{
					name: "uintptr",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, uintptr](Registry{}, Scoped)
					},
					expectedErr: NoDefaultFactory{
						Type: reflect.TypeFor[uintptr](),
					},
				},
				{
					name: "func",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, func()](Registry{}, Scoped)
					},
					expectedErr: NoDefaultFactory{
						Type: reflect.TypeFor[func()](),
					},
				},
				{
					name: "unsafe.Pointer",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, unsafe.Pointer](Registry{}, Scoped)
					},
					expectedErr: NoDefaultFactory{
						Type: reflect.TypeFor[unsafe.Pointer](),
					},
				},
				{
					name: "*uintptr",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, *uintptr](Registry{}, Scoped)
					},
					expectedErr: NoDefaultFactory{
						Type: reflect.TypeFor[*uintptr](),
					},
				},
				{
					name: "**func",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, **func()](Registry{}, Scoped)
					},
					expectedErr: NoDefaultFactory{
						Type: reflect.TypeFor[**func()](),
					},
				},
				{
					name: "***unsafe.Pointer",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, ***unsafe.Pointer](Registry{}, Scoped)
					},
					expectedErr: NoDefaultFactory{
						Type: reflect.TypeFor[***unsafe.Pointer](),
					},
				},
			}

			for _, tt := range testCases {
				t.Run(fmt.Sprintf("returns NoDefaultFactory for %s", tt.name), func(t *testing.T) {
					_, err := tt.fn()
					if !errors.Is(err, ErrNoDefaultFactory) {
						t.Fatalf("expected %q; got %q", ErrNoDefaultFactory, err)
					}
					var noDefaultFactory NoDefaultFactory
					if !errors.As(err, &noDefaultFactory) {
						t.Fatalf("expected %v to be %T", err, noDefaultFactory)
					}
					if type_ := tt.expectedErr.Type; noDefaultFactory.Type != type_ {
						t.Errorf("expected err.Type to be %v; got %v", type_, noDefaultFactory.Type)
					}
				})
			}
		})

		t.Run("returns UndefinedLifetime when lifetime is undefined", func(t *testing.T) {
			undefinedValue := Lifetime(13)
			_, err := RegisterType[interface{}, struct{}](Registry{}, undefinedValue)
			if !errors.Is(err, ErrUndefinedLifetime) {
				t.Fatalf("expected %q; got %q", ErrUndefinedLifetime, err)
			}
			var undefinedLifetime UndefinedLifetime
			if !errors.As(err, &undefinedLifetime) {
				t.Fatalf("expected %v to be %T", err, undefinedLifetime)
			}
			if undefinedLifetime.Value != undefinedValue {
				t.Errorf("expected err.Value to be %v; got %v", undefinedValue, undefinedLifetime.Value)
			}
		})

		t.Run("unsharable types", func(t *testing.T) {
			testCases := []struct {
				name        string
				fn          func() (Registry, error)
				expectedErr UnsharableType
			}{
				{
					name: "scoped struct",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, struct{}](Registry{}, Scoped)
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[struct{}](),
						Lifetime: Scoped,
					},
				},
				{
					name: "singleton struct",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, struct{}](Registry{}, Singleton)
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[struct{}](),
						Lifetime: Singleton,
					},
				},
				{
					name: "scoped array",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, [3]int](Registry{}, Scoped)
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[[3]int](),
						Lifetime: Scoped,
					},
				},
				{
					name: "singleton array",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, [3]int](Registry{}, Singleton)
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[[3]int](),
						Lifetime: Singleton,
					},
				},
				{
					name: "scoped slice",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, []int](Registry{}, Scoped)
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[[]int](),
						Lifetime: Scoped,
					},
				},
				{
					name: "singleton slice",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, []int](Registry{}, Singleton)
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[[]int](),
						Lifetime: Singleton,
					},
				},
				{
					name: "scoped map",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, map[int]string](Registry{}, Scoped)
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[map[int]string](),
						Lifetime: Scoped,
					},
				},
				{
					name: "singleton map",
					fn: func() (Registry, error) {
						return RegisterType[interface{}, map[int]string](Registry{}, Singleton)
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[map[int]string](),
						Lifetime: Singleton,
					},
				},
			}

			for _, tt := range testCases {
				t.Run(fmt.Sprintf("returns UnsharableType for %s", tt.name), func(t *testing.T) {
					_, err := tt.fn()
					if !errors.Is(err, ErrUnsharableType) {
						t.Fatalf("expected %q; got %q", ErrUnsharableType, err)
					}
					var unsharableType UnsharableType
					if !errors.As(err, &unsharableType) {
						t.Fatalf("expected %v to be %T", err, unsharableType)
					}
					if type_ := tt.expectedErr.Type; unsharableType.Type != type_ {
						t.Errorf("expected err.Type to be %v; got %v", type_, unsharableType.Type)
					}
					if lifetime := tt.expectedErr.Lifetime; unsharableType.Lifetime != lifetime {
						t.Errorf("expected err.Lifetime to be %v; got %v", lifetime, unsharableType.Lifetime)
					}
				})
			}
		})

		t.Run("default factories", func(t *testing.T) {

			testCases := []struct {
				name string
				fn   func() (Registry, error)
			}{
				{
					name: "int",
					fn: func() (Registry, error) {
						return RegisterType[int, int](Registry{}, Transient)
					},
				},
				{
					name: "string",
					fn: func() (Registry, error) {
						return RegisterType[string, string](Registry{}, Transient)
					},
				},
				{
					name: "struct",
					fn: func() (Registry, error) {
						return RegisterType[struct{}, struct{}](Registry{}, Transient)
					},
				},
				{
					name: "array",
					fn: func() (Registry, error) {
						return RegisterType[[3]int, [3]int](Registry{}, Transient)
					},
				},
				{
					name: "slice",
					fn: func() (Registry, error) {
						return RegisterType[[]int, []int](Registry{}, Transient)
					},
				},
				{
					name: "map",
					fn: func() (Registry, error) {
						return RegisterType[map[int]string, map[int]string](Registry{}, Transient)
					},
				},
			}

			for _, tt := range testCases {
				t.Run(fmt.Sprintf("does not return error for transient %s", tt.name), func(t *testing.T) {
					_, err := tt.fn()
					if err != nil {
						t.Fatalf("unexpected error %v", err)
					}
				})
			}
		})

		t.Run("sharable types", func(t *testing.T) {

			testCases := []struct {
				name string
				fn   func() (Registry, error)
			}{
				{
					name: "scoped *struct",
					fn: func() (Registry, error) {
						return RegisterType[*struct{}, *struct{}](Registry{}, Scoped)
					},
				},
				{
					name: "singleton *struct",
					fn: func() (Registry, error) {
						return RegisterType[*struct{}, *struct{}](Registry{}, Singleton)
					},
				},
				{
					name: "scoped chan",
					fn: func() (Registry, error) {
						return RegisterType[chan int, chan int](Registry{}, Scoped)
					},
				},
				{
					name: "singleton chan",
					fn: func() (Registry, error) {
						return RegisterType[chan int, chan int](Registry{}, Singleton)
					},
				},
			}

			for _, tt := range testCases {
				t.Run(fmt.Sprintf("does not return error for %s", tt.name), func(t *testing.T) {
					_, err := tt.fn()
					if err != nil {
						t.Fatalf("unexpected error %v", err)
					}
				})
			}
		})
	})

	t.Run("RegisterFactory", func(t *testing.T) {

		t.Run("returns NonConcreteImplementation when Impl is an interface", func(t *testing.T) {
			_, err := RegisterFactory[io.Reader](Registry{}, Transient, func(r Resolver) (io.ReadWriter, error) {
				return bytes.NewBuffer([]byte{}), nil
			})
			if !errors.Is(err, ErrNonConcreteImplementation) {
				t.Fatalf("expected %q; got %q", ErrNonConcreteImplementation, err)
			}
			var nonConcreteImpl NonConcreteImplementation
			if !errors.As(err, &nonConcreteImpl) {
				t.Fatalf("expected %v to be %T", err, nonConcreteImpl)
			}
			if type_ := reflect.TypeFor[io.ReadWriter](); nonConcreteImpl.Type != type_ {
				t.Errorf("expected err.Type to be %v; got %v", type_, nonConcreteImpl.Type)
			}
		})

		t.Run("returns InvalidImplementation when Impl cannot be assigned to Target", func(t *testing.T) {
			_, err := RegisterFactory[string](Registry{}, Transient, func(Resolver) (struct{}, error) {
				return struct{}{}, nil
			})
			if !errors.Is(err, ErrInvalidImplementation) {
				t.Fatalf("expected %q; got %q", ErrInvalidImplementation, err)
			}
			var invalidImpl InvalidImplementation
			if !errors.As(err, &invalidImpl) {
				t.Fatalf("expected %v to be %T", err, invalidImpl)
			}
			if strType := reflect.TypeFor[string](); invalidImpl.Target != strType {
				t.Errorf("expected err.Target to be %v; got %v", strType, invalidImpl.Target)
			}
			if structType := reflect.TypeFor[struct{}](); invalidImpl.Type != structType {
				t.Errorf("expected err.Impl to be %v; got %v", structType, invalidImpl.Type)
			}
		})

		t.Run("returns UndefinedLifetime when lifetime is undefined", func(t *testing.T) {
			undefinedValue := Lifetime(13)
			_, err := RegisterFactory[interface{}](Registry{}, undefinedValue, func(Resolver) (struct{}, error) {
				return struct{}{}, nil
			})
			if !errors.Is(err, ErrUndefinedLifetime) {
				t.Fatalf("expected %q; got %q", ErrUndefinedLifetime, err)
			}
			var undefinedLifetime UndefinedLifetime
			if !errors.As(err, &undefinedLifetime) {
				t.Fatalf("expected %v to be %T", err, undefinedLifetime)
			}
			if undefinedLifetime.Value != undefinedValue {
				t.Errorf("expected err.Value to be %v; got %v", undefinedValue, undefinedLifetime.Value)
			}
		})

		t.Run("unsharable types", func(t *testing.T) {
			testCases := []struct {
				name        string
				fn          func() (Registry, error)
				expectedErr UnsharableType
			}{
				{
					name: "scoped struct",
					fn: func() (Registry, error) {
						return RegisterFactory[interface{}](Registry{}, Scoped, func(Resolver) (struct{}, error) {
							return struct{}{}, nil
						})
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[struct{}](),
						Lifetime: Scoped,
					},
				},
				{
					name: "singleton struct",
					fn: func() (Registry, error) {
						return RegisterFactory[interface{}](Registry{}, Singleton, func(Resolver) (struct{}, error) {
							return struct{}{}, nil
						})
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[struct{}](),
						Lifetime: Singleton,
					},
				},
				{
					name: "scoped array",
					fn: func() (Registry, error) {
						return RegisterFactory[interface{}](Registry{}, Scoped, func(Resolver) ([3]int, error) {
							return [3]int{}, nil
						})
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[[3]int](),
						Lifetime: Scoped,
					},
				},
				{
					name: "singleton array",
					fn: func() (Registry, error) {
						return RegisterFactory[interface{}](Registry{}, Singleton, func(Resolver) ([3]int, error) {
							return [3]int{}, nil
						})
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[[3]int](),
						Lifetime: Singleton,
					},
				},
				{
					name: "scoped slice",
					fn: func() (Registry, error) {
						return RegisterFactory[interface{}](Registry{}, Scoped, func(Resolver) ([]int, error) {
							return []int{}, nil
						})
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[[]int](),
						Lifetime: Scoped,
					},
				},
				{
					name: "singleton slice",
					fn: func() (Registry, error) {
						return RegisterFactory[interface{}](Registry{}, Singleton, func(Resolver) ([]int, error) {
							return []int{}, nil
						})
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[[]int](),
						Lifetime: Singleton,
					},
				},
				{
					name: "scoped map",
					fn: func() (Registry, error) {
						return RegisterFactory[interface{}](Registry{}, Scoped, func(Resolver) (map[int]string, error) {
							return map[int]string{}, nil
						})
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[map[int]string](),
						Lifetime: Scoped,
					},
				},
				{
					name: "singleton map",
					fn: func() (Registry, error) {
						return RegisterFactory[interface{}](Registry{}, Singleton, func(Resolver) (map[int]string, error) {
							return map[int]string{}, nil
						})
					},
					expectedErr: UnsharableType{
						Type:     reflect.TypeFor[map[int]string](),
						Lifetime: Singleton,
					},
				},
			}

			for _, tt := range testCases {
				t.Run(fmt.Sprintf("returns UnsharableType for %s", tt.name), func(t *testing.T) {
					_, err := tt.fn()
					if !errors.Is(err, ErrUnsharableType) {
						t.Fatalf("expected %q; got %q", ErrUnsharableType, err)
					}
					var unsharableType UnsharableType
					if !errors.As(err, &unsharableType) {
						t.Fatalf("expected %v to be %T", err, unsharableType)
					}
					if type_ := tt.expectedErr.Type; unsharableType.Type != type_ {
						t.Errorf("expected err.Type to be %v; got %v", type_, unsharableType.Type)
					}
					if lifetime := tt.expectedErr.Lifetime; unsharableType.Lifetime != lifetime {
						t.Errorf("expected err.Lifetime to be %v; got %v", lifetime, unsharableType.Lifetime)
					}
				})
			}
		})

		t.Run("returns NilFactory when factory is nil", func(t *testing.T) {
			_, err := RegisterFactory[interface{}, struct{}](Registry{}, Transient, nil)
			if !errors.Is(err, ErrNilFactory) {
				t.Fatalf("expected %q; got %q", ErrNilFactory, err)
			}
		})

		t.Run("transient types", func(t *testing.T) {

			testCases := []struct {
				name string
				fn   func() (Registry, error)
			}{
				{
					name: "int",
					fn: func() (Registry, error) {
						return RegisterFactory[int, int](Registry{}, Transient, func(r Resolver) (int, error) {
							return 7, nil
						})
					},
				},
				{
					name: "string",
					fn: func() (Registry, error) {
						return RegisterFactory[string, string](Registry{}, Transient, func(r Resolver) (string, error) {
							return "seven", nil
						})
					},
				},
				{
					name: "struct",
					fn: func() (Registry, error) {
						return RegisterFactory[struct{}, struct{}](Registry{}, Transient, func(r Resolver) (struct{}, error) {
							return struct{}{}, nil
						})
					},
				},
				{
					name: "array",
					fn: func() (Registry, error) {
						return RegisterFactory[[3]int, [3]int](Registry{}, Transient, func(r Resolver) ([3]int, error) {
							return [3]int{1, 1, 1}, nil
						})
					},
				},
				{
					name: "slice",
					fn: func() (Registry, error) {
						return RegisterFactory[[]int, []int](Registry{}, Transient, func(r Resolver) ([]int, error) {
							return []int{4, 2, 1}, nil
						})
					},
				},
				{
					name: "map",
					fn: func() (Registry, error) {
						return RegisterFactory[map[int]string, map[int]string](Registry{}, Transient, func(r Resolver) (map[int]string, error) {
							return make(map[int]string), nil
						})
					},
				},
			}

			for _, tt := range testCases {
				t.Run(fmt.Sprintf("does not return error for transient %s", tt.name), func(t *testing.T) {
					_, err := tt.fn()
					if err != nil {
						t.Fatalf("unexpected error %v", err)
					}
				})
			}
		})

		t.Run("sharable types", func(t *testing.T) {

			testCases := []struct {
				name string
				fn   func() (Registry, error)
			}{
				{
					name: "scoped *struct",
					fn: func() (Registry, error) {
						return RegisterFactory[*struct{}](Registry{}, Scoped, func(r Resolver) (*struct{}, error) {
							return &struct{}{}, nil
						})
					},
				},
				{
					name: "singleton *struct",
					fn: func() (Registry, error) {
						return RegisterFactory[*struct{}](Registry{}, Singleton, func(r Resolver) (*struct{}, error) {
							return &struct{}{}, nil
						})
					},
				},
				{
					name: "scoped chan",
					fn: func() (Registry, error) {
						return RegisterFactory[chan int](Registry{}, Scoped, func(r Resolver) (chan int, error) {
							return make(chan int), nil
						})
					},
				},
				{
					name: "singleton chan",
					fn: func() (Registry, error) {
						return RegisterFactory[chan int](Registry{}, Singleton, func(r Resolver) (chan int, error) {
							return make(chan int), nil
						})
					},
				},
			}

			for _, tt := range testCases {
				t.Run(fmt.Sprintf("does not return error for %s", tt.name), func(t *testing.T) {
					_, err := tt.fn()
					if err != nil {
						t.Fatalf("unexpected error %v", err)
					}
				})
			}
		})
	})
}
