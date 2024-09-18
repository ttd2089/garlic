package di

import (
	"context"
	"reflect"
	"sync"
)

// A Scope is a [Provider] that can resolve [Scoped] values in addition to [Transient] and
// [Singleton] values. A Scope will create a single instance of a value for a type registered
type Scope struct {
	root         RootProvider
	scopedValues *instanceMap
}

// NewScope creates a new [Scope] which can resolve [Scoped] values as well as [Transient]
// and [Singleton] values.
func (scope Scope) NewScope() Scope {
	return scope.root.NewScope()
}

// Resolve returns an instance of the requested type if it was registered.
func (scope Scope) Resolve(typ reflect.Type) (any, error) {
	registration, ok := scope.root.registrations[typ]
	if ok && registration.lifetime == Scoped {
		return scope.scopedValues.resolve(typ, registration.factory, scope)
	}
	return scope.root.Resolve(typ)
}

// A ContextCloser is a value that can be closed with a [context.Context].
type ContextCloser interface {
	Close(context.Context) error
}

// A Closer is a value that can be closed.
type Closer interface {
	Close() error
}

func (scope Scope) Close(ctx context.Context) []error {

	values := scope.scopedValues.values()
	contextClosers := make([]ContextCloser, 0, len(values))
	closers := make([]Closer, 0, len(values))
	for _, value := range values {
		if contextCloser, ok := value.(ContextCloser); ok {
			contextClosers = append(contextClosers, contextCloser)
			continue
		}
		if closer, ok := value.(Closer); ok {
			closers = append(closers, closer)
		}
	}

	n := len(contextClosers) + len(closers)
	closeErrorsCh := make(chan error, n)
	closeErrors := make([]error, 0, n)

	wg := sync.WaitGroup{}
	wg.Add(n)
	wgDone := make(chan struct{})
	go func() {
		defer close(wgDone)
		wg.Wait()
	}()

	for _, contextCloser := range contextClosers {
		go func() {
			defer wg.Done()
			closeErrorsCh <- contextCloser.Close(ctx)
		}()
	}

	for _, closer := range closers {
		go func() {
			defer wg.Done()
			closeErrorsCh <- closer.Close()
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return closeErrors
		case <-wgDone:
			return closeErrors
		case err := <-closeErrorsCh:
			if err != nil {
				closeErrors = append(closeErrors, err)
			}
		}
	}
}
