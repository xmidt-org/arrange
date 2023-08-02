/**
 * Copyright 2023 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package arrangehttp

import "go.uber.org/multierr"

// Option represents something that can modify a target object.
type Option[T any] interface {
	Apply(*T) error
}

// OptionFunc is a closure type that can act as an Option.
type OptionFunc[T any] func(*T) error

func (of OptionFunc[T]) Apply(t *T) error {
	return of(t)
}

// Options is an aggregate Option that allows several options to
// be grouped together.
type Options[T any] []Option[T]

// Apply applies all the options in this slice, returning an
// aggregate error if any errors occurred.
func (o Options[T]) Apply(t *T) (err error) {
	for _, opt := range o {
		err = multierr.Append(err, opt.Apply(t))
	}

	return
}

// OptionClosure represents the closure types that are convertible
// into Option objects.
type OptionClosure[T any] interface {
	~func(*T) | ~func(*T) error
}

// AsOption converts a closure into an Option for a given target type.
func AsOption[T any, F OptionClosure[T]](f F) Option[T] {
	fv := any(f)
	if of, ok := fv.(func(*T) error); ok {
		return OptionFunc[T](of)
	}

	return OptionFunc[T](func(t *T) error {
		fv.(func(*T))(t)
		return nil
	})
}

// ApplyOptions applies several options to a target.  This function
// returns the original target t so that it can be used with fx.Decorate.
func ApplyOptions[T any](t *T, opts ...Option[T]) (result *T, err error) {
	result = t
	err = Options[T](opts).Apply(result)
	return
}

// InvalidOption returns an Option that returns the given error.
// Useful instead of nil or a panic to indicate that something in the setup
// of an Option went wrong.
func InvalidOption[T any](err error) Option[T] {
	return OptionFunc[T](func(_ *T) error {
		return err
	})
}
