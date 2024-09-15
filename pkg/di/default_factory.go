package di

import (
	"errors"
	"reflect"
)

// GetDefaultFactory returns the default factory for the requested type, or [ErrNoDefaultFactory]
// if the type has no default factory.
func GetDefaultFactory[T any]() (Factory[T], error) {
	typ := reflect.TypeFor[T]()
	factory, err := getDefaultFactory(typ)
	if err != nil {
		return nil, err
	}
	return func(r Resolver) (T, error) {
		v, err := factory(r)
		if err != nil {
			var zero T
			return zero, err
		}
		return v.(T), nil
	}, nil
}

func getDefaultFactory(typ reflect.Type) (factoryFunc, error) {
	switch typ.Kind() {
	case
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.Array,
		reflect.String,
		reflect.Slice:
		return func(Resolver) (any, error) {
			return reflect.Zero(typ).Interface(), nil
		}, nil
	case reflect.Map:
		return func(Resolver) (any, error) {
			return reflect.MakeMap(typ).Interface(), nil
		}, nil
	case reflect.Chan:
		return func(Resolver) (any, error) {
			return reflect.MakeChan(typ, 0).Interface(), nil
		}, nil
	case reflect.Struct:
		return getDefaultStructFactory(typ)
	case reflect.Pointer:
		return getDefaultPointerFactory(typ)
	}

	return nil, NoDefaultFactory{
		Type: typ,
	}
}

func getDefaultStructFactory(typ reflect.Type) (factoryFunc, error) {
	return func(r Resolver) (any, error) {
		val := reflect.New(typ)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			resolved, err := r.Resolve(field.Type)
			if err != nil {
				return nil, resolverError{wrapped: err}
			}
			resolvedType := reflect.TypeOf(resolved)
			if resolvedType == nil || !resolvedType.AssignableTo(field.Type) {
				return nil, InvalidResolution{
					Requested: field.Type,
					Returned:  reflect.TypeOf(resolved),
				}
			}
			val.Elem().Field(i).Set(reflect.ValueOf(resolved))
		}
		return val.Elem().Interface(), nil
	}, nil
}

func getDefaultPointerFactory(typ reflect.Type) (factoryFunc, error) {
	elemFactory, err := getDefaultFactory(typ.Elem())
	if errors.Is(err, ErrNoDefaultFactory) {
		return nil, NoDefaultFactory{
			Type: typ,
		}
	}
	if err != nil {
		return nil, err
	}
	return func(r Resolver) (any, error) {
		v, err := elemFactory(r)
		if err != nil {
			return nil, err
		}
		pVal := reflect.New(typ.Elem())
		pVal.Elem().Set(reflect.ValueOf(v))
		return pVal.Interface(), nil
	}, nil
}
