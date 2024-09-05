package di

// A Lifetime expresses the conditions under which an instance of a type will be instantiated or
// reused across distinct resolutions.
type Lifetime int

const (
	// Transient means that a new instance is created every time the type is resolved.
	Transient Lifetime = iota + 1

	// Scoped means that the type is instantiated once per resolution scope and the same instance
	// is returned every time the type is resolved in a given scope.
	//
	// NOTE: The [Scoped] [Lifetime] can only be used for [Sharable Types].
	//
	// [Sharable Types]: https://github.com/ttd2089/garlic?tab=readme-ov-file#sharable-types
	Scoped

	// Singleton means that the type is only ever instantiated once and the same instance is
	// returned every time the type is resolved across all resolution scopes.
	//
	// NOTE: The [Singleton] [Lifetime] can only be used for [Sharable Types].
	//
	// [Sharable Types]: https://github.com/ttd2089/garlic?tab=readme-ov-file#sharable-types
	Singleton
)

var knownLifetimes map[Lifetime]string = map[Lifetime]string{
	Transient: "Transient",
	Scoped:    "Scoped",
	Singleton: "Singleton",
}

func (lifetime Lifetime) String() string {
	if name, ok := knownLifetimes[lifetime]; ok {
		return name
	}
	return "Unknown"
}
