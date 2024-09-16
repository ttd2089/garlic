package di

import (
	"errors"
	"reflect"
	"testing"
)

func TestResolve(t *testing.T) {

	t.Run("returns ErrNilResolver when Resolver is nil", func(t *testing.T) {
		if _, err := Resolve[interface{}](nil); !errors.Is(err, ErrNilResolver) {
			t.Fatalf("expected %v; got %v", ErrNilResolver, err)
		}
	})

	t.Run("requests an instance of the type T from the Resolver", func(t *testing.T) {
		expected := reflect.TypeFor[interface{}]()
		resolver := mockResolver{}
		resolver.returns(&struct{}{}, nil)
		_, _ = Resolve[interface{}](&resolver)
		if resolver.requestedTypes[0] != expected {
			t.Errorf("expected %v; got %v", expected, resolver.requestedTypes[0])
		}
	})

	t.Run("returns ErrResolverError when the Resolver returns an error", func(t *testing.T) {
		expectedErr := errors.New("expected error")
		resolver := mockResolver{}
		resolver.returns(struct{}{}, expectedErr)
		_, actualErr := Resolve[struct{}](&resolver)
		if !errors.Is(actualErr, ErrResolverError) {
			t.Errorf("expected %v; got %v", expectedErr, ErrResolverError)
		}
		if !errors.Is(actualErr, expectedErr) {
			t.Errorf("expected %v; got %v", expectedErr, actualErr)
		}
	})

	t.Run("returns ErrInvalidResolution when the returned value is not assignable to requested to type", func(t *testing.T) {
		resolver := mockResolver{}
		resolver.returns(struct{}{}, nil)
		_, err := Resolve[string](&resolver)
		if !errors.Is(err, ErrInvalidResolution) {
			t.Errorf("expected %v; got %v", ErrInvalidResolution, err)
		}
		var invalidType InvalidResolution
		if !errors.As(err, &invalidType) {
			t.Fatalf("expected %v to be %T", err, invalidType)
		}
		if strType := reflect.TypeFor[string](); invalidType.Requested != strType {
			t.Errorf("expected err.Actual to be %v; got %v", strType, invalidType.Requested)
		}
		if structType := reflect.TypeFor[struct{}](); invalidType.Returned != structType {
			t.Errorf("expected err.Actual to be %v; got %v", structType, invalidType.Returned)
		}
	})

	t.Run("returns the resolved value when its assignable to requested type", func(t *testing.T) {
		expected := &struct{}{}
		resolver := mockResolver{}
		resolver.returns(expected, nil)
		actual, err := Resolve[interface{}](&resolver)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if actual != expected {
			t.Errorf("expected %v; got %v", expected, actual)
		}
	})
}

type mockResolver struct {
	returnValues []struct {
		v   any
		err error
	}
	requestedTypes []reflect.Type
}

func (mock *mockResolver) returns(v any, err error) {
	mock.returnValues = append(mock.returnValues, struct {
		v   any
		err error
	}{
		v:   v,
		err: err,
	})
}

func (mock *mockResolver) Resolve(typ reflect.Type) (any, error) {
	mock.requestedTypes = append(mock.requestedTypes, typ)
	if len(mock.returnValues) == 0 {
		panic("no return values configured")
	}
	r := mock.returnValues[0]
	mock.returnValues = mock.returnValues[1:]
	return r.v, r.err
}
