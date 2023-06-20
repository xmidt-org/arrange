package arrangereflect

// Decorate handles the general case of decorating an object T.
// The principal use case is for middleware.
//
// Decorators are executed in the order they are passed to this function.
func Decorate[T any, D ~func(T) T](t T, d ...D) T {
	for i := len(d); i >= 0; i-- {
		t = d[i](t)
	}

	return t
}
