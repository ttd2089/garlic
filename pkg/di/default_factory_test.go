package di

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

// asFactoryFunc returns the result of GetDefaultFactory as a factoryFunc. This enables us to write
// cases for GetDefaultFactory with differing type parameters in a single set of table test cases.
func asFactoryFunc[T any](in func() (Factory[T], error)) func() (factoryFunc, error) {
	return func() (factoryFunc, error) {
		factory, err := in()
		if err != nil {
			return nil, err
		}
		return func(r Resolver) (any, error) {

			value, err := factory(r)
			if err != nil {
				var zero T
				return zero, err
			}
			return value, nil
		}, nil
	}
}

func Test_getDefaultFactory(t *testing.T) {

	t.Run("no default factory", func(t *testing.T) {

		testCases := []struct {
			getDefaultFactory func() (factoryFunc, error)
			typ               reflect.Type
		}{
			{
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[uintptr]),
				typ:               reflect.TypeFor[uintptr](),
			},
			{
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[func()]),
				typ:               reflect.TypeFor[func()](),
			},
			{
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[unsafe.Pointer]),
				typ:               reflect.TypeFor[unsafe.Pointer](),
			},
			{
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[*uintptr]),
				typ:               reflect.TypeFor[*uintptr](),
			},
			{
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[*func()]),
				typ:               reflect.TypeFor[*func()](),
			},
			{
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[*unsafe.Pointer]),
				typ:               reflect.TypeFor[*unsafe.Pointer](),
			},
		}

		for _, tt := range testCases {
			t.Run(fmt.Sprintf("returns NoDefaultFactory for %T", tt.typ), func(t *testing.T) {
				_, err := tt.getDefaultFactory()
				if !errors.Is(err, ErrNoDefaultFactory) {
					t.Fatalf("expected %q; got %q", ErrNoDefaultFactory, err)
				}
				var noDefaultFactory NoDefaultFactory
				if !errors.As(err, &noDefaultFactory) {
					t.Fatalf("expected %v to be %T", err, noDefaultFactory)
				}
				if typ := tt.typ; noDefaultFactory.Type != typ {
					t.Errorf("expected err.Type to be %v; got %v", typ, noDefaultFactory.Type)
				}
			})
		}
	})

	t.Run("default factory for", func(t *testing.T) {

		type decision bool

		testCases := []struct {
			name              string
			getDefaultFactory func() (factoryFunc, error)
			typ               reflect.Type
			expected          any
		}{
			{
				name:              "bool returns false",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[bool]),
				expected:          false,
			},
			{
				name:              "pointer to bool returns pointer to false",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[*bool]),
				expected: func() *bool {
					b := false
					return &b
				}(),
			},
			{
				name:              "pointer to pointer to bool returns pointer to pointer to false",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[**bool]),
				expected: func() **bool {
					b := false
					p := &b
					return &p
				}(),
			},
			{
				name:              "type defined as bool returns false",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[decision]),
				expected:          decision(false),
			},
			{
				name:              "int returns zero",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[int]),
				expected:          int(0),
			},
			{
				name:              "array returns array of zero-values",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[[3]int]),
				expected:          [3]int{},
			}, {
				name:              "pointer to array returns pointer to array of zero-values",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[*[3]int]),
				expected:          &[3]int{},
			},
			{
				name:              "pointer to string returns pointer to \"\"",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[*string]),
				expected: func() *string {
					s := ""
					return &s
				}(),
			},
			{
				name:              "slice returns nil",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[[]int]),
				expected: func() []int {
					return nil
				}(),
			},
			{
				name:              "pointer to slice returns non-nil pointer to nil",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[*[]int]),
				expected: func() *[]int {
					var s []int
					return &s
				}(),
			},
			{
				name:              "map returns empty map",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[map[int]string]),
				expected:          make(map[int]string),
			},
			{
				name:              "pointer to map returns pointer to empty map",
				getDefaultFactory: asFactoryFunc(GetDefaultFactory[*map[int]string]),
				expected:          &map[int]string{},
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				factory, err := tt.getDefaultFactory()
				if err != nil {
					t.Fatalf("unexpected error getting factory: %v", err)
				}
				v, err := factory(nil)
				if err != nil {
					t.Fatalf("unexpected error from factory: %v", err)
				}
				if !reflect.DeepEqual(v, tt.expected) {
					t.Fatalf("expected %#[1]v (%[1]T); got %#[2]v (%[2]T)", tt.expected, v)
				}
			})
		}

		// Identical channels aren't considered equal so we needed a custom body for this case.
		t.Run("chan returns unbuffered chan", func(t *testing.T) {
			factory, err := GetDefaultFactory[chan int]()
			if err != nil {
				t.Fatalf("unexpected error getting factory: %v", err)
			}
			ch, err := factory(nil)
			if err != nil {
				t.Fatalf("unexpected error from factory: %v", err)
			}
			if len := cap(ch); len != 0 {
				t.Fatalf("expected unbuffered chan; got %d-element buffer", len)
			}
		})

		t.Run("struct returns instance of struct", func(t *testing.T) {

			factory, err := GetDefaultFactory[struct{}]()
			if err != nil {
				t.Fatalf("unexpected error getting factory: %v", err)
			}
			_, err = factory(nil)
			if err != nil {
				t.Fatalf("unexpected error from factory: %v", err)
			}
		})

		t.Run("struct initializes exported fields with resolver", func(t *testing.T) {

			expectedWidget := widget{
				X: 42,
				y: 1.618,
			}
			expectedGadget := gadget{
				Names: map[int]string{
					1: "one",
					2: "two",
				},
				counts: map[string]int{
					"three": 3,
					"four":  4,
				},
			}

			resolver := testResolver{
				resolutions: map[reflect.Type]testResolverResolution{
					reflect.TypeFor[widget](): {
						val: expectedWidget,
					},
					reflect.TypeFor[*gadget](): {
						val: &expectedGadget,
					},
				},
			}

			expected := thing{
				Widget: expectedWidget,
				Gadget: &expectedGadget,
			}

			factory, _ := GetDefaultFactory[thing]()

			thing, err := factory(resolver)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(thing, expected) {
				t.Fatalf("expected %v; got %v", expected, thing)
			}
		})

		t.Run("struct does not initialize unexported fields", func(t *testing.T) {
			expected := unexportedFieldsOnly{}
			factory, _ := GetDefaultFactory[unexportedFieldsOnly]()

			ufo, err := factory(testResolver{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(ufo, expected) {
				t.Fatalf("expected %v; got %v", expected, ufo)
			}
		})

		t.Run("struct does not initialize exported fields recursively", func(t *testing.T) {

			unexpectedWidget := widget{
				X: 13,
				y: 3.14,
			}
			unexpectedGadget := gadget{
				Names: map[int]string{
					11: "eleven",
					12: "twelve",
				},
				counts: map[string]int{
					"thirteen": 13,
					"fourteen": 14,
				},
			}

			expectedThing := thing{
				Widget: widget{
					X: 42,
					y: 1.618,
				},
				Gadget: &gadget{
					Names: map[int]string{
						1: "one",
						2: "two",
					},
					counts: map[string]int{
						"three": 3,
						"four":  4,
					},
				},
			}

			expected := recursiveStruct{
				Thing: expectedThing,
			}

			resolver := testResolver{
				resolutions: map[reflect.Type]testResolverResolution{
					reflect.TypeFor[widget](): {
						val: unexpectedWidget,
					},
					reflect.TypeFor[*gadget](): {
						val: &unexpectedGadget,
					},
					reflect.TypeFor[thing](): {
						val: expectedThing,
					},
				},
			}

			factory, _ := GetDefaultFactory[recursiveStruct]()

			rs, _ := factory(resolver)

			if !reflect.DeepEqual(rs, expected) {
				t.Fatalf("expected %v; got %v", expected, rs)
			}

		})

		t.Run("pointer to struct returns non-nil pointer to struct", func(t *testing.T) {

			factory, err := GetDefaultFactory[*unexportedFieldsOnly]()
			if err != nil {
				t.Fatalf("unexpected error getting factory: %v", err)
			}
			pufo, err := factory(testResolver{})
			if err != nil {
				t.Fatalf("unexpected error from factory: %v", err)
			}
			if pufo == nil {
				t.Fatalf("expected non-nil %[1]T; got %[1]v", pufo)
			}
		})

	})
}

type testResolver struct {
	resolutions map[reflect.Type]testResolverResolution
}

type testResolverResolution struct {
	val any
	err error
}

func (r testResolver) Resolve(typ reflect.Type) (any, error) {
	if v, ok := r.resolutions[typ]; ok {
		return v.val, v.err
	}
	return nil, fmt.Errorf("unexpected call: testResolver.Resolve(%v)", typ)
}

type widget struct {
	X int
	y float64
}

type gadget struct {
	Names  map[int]string
	counts map[string]int
}

type thing struct {
	Widget widget
	Gadget *gadget
}

type unexportedFieldsOnly struct {
	//lint:ignore U1000 Testing that unexported fields are not initialized.
	widget widget
	//lint:ignore U1000 Testing that unexported fields are not initialized.
	gadget *gadget
}

type recursiveStruct struct {
	Thing thing
}
