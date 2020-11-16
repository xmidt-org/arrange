package arrange

import (
	"fmt"
	"reflect"
	"strconv"

	"go.uber.org/dig"
)

// Dependency represents a reflected value (possibly) injected by an enclosing fx.App
type Dependency struct {
	// Container is the struct in which this dependency occurred.
	//
	// This field is only set if the injected value was part of an enclosing struct
	// that was populated by an fx.App.
	Container reflect.Type

	// Field is the struct field from which this dependency was taken.  This will
	// be nil for global, unnamed components.
	Field *reflect.StructField

	// Value is the actual value that was injected.  For plain dependencies that
	// were not part of an fx.In struct, this will be the only field set.
	Value reflect.Value
}

// TagValue returns the given metatag value for this dependency.  This method will
// return the empty string for all keys if this dependency didn't come from a struct.
func (d Dependency) TagValue(key string) (v string) {
	if d.Field != nil {
		v = d.Field.Tag.Get(key)
	}

	return
}

// Name returns the component name for this dependency.  This will always
// return the empty string if this dependency didn't come from a struct.
func (d Dependency) Name() string {
	return d.TagValue("name")
}

// Group returns the value group name for this dependency.  This will always
// return the empty string if this dependency didn't come from a struct.
func (d Dependency) Group() string {
	return d.TagValue("group")
}

// Optional returns whether this component can be missing in the enclosing fx.App.
// This will always return false (i.e. required) if this dependency didn't come
// from a struct.
func (d Dependency) Optional() (v bool) {
	v, _ = strconv.ParseBool(d.TagValue("optional"))
	return
}

// Injected returns true if this dependency was actually injected.  This
// method returns false if both d.Optional is true and the value represents
// the zero value.
//
// Note that this method can give false negatives for non-pointer dependencies.
// If an optional component is present but is set to the zero value, this method
// will still return false.  Callers should be aware of this case and implement
// application-specific logic where necessary.
func (d Dependency) Injected() bool {
	return !d.Optional() || !d.Value.IsZero()
}

// String returns a human readable representation of this dependency
func (d Dependency) String() string {
	if d.Container != nil && d.Field != nil {
		return fmt.Sprintf("%s.%s %s", d.Container, d.Field.Name, d.Field.Tag)
	}

	return fmt.Sprintf("%s", d.Value.Type())
}

// DependencyVisitor is a visitor predicate used by VisitDependencies as a callback
// for each dependency of a set.  If this method returns false, visitation will be
// halted early.
type DependencyVisitor func(Dependency) bool

// VisitDependencies applies the given visitor to a sequence of dependencies.  The deps
// slice can contain any values allowed by go.uber.org/fx in constructor functions, i.e.
// they must all be dependencies that were either injected or skipped (as when optional:"true" is set).
//
// If any value in deps is a struct that embeds fx.In, then that struct's fields are walked
// recursively.  Any exported fields are assumed to have been injected (or, skipped), and the visitor
// is invoked for each of them.
//
// For non-struct values or for structs that do not embed fx.In, the visitor is simply invoked
// with that value but with Name, Group, etc fields left unset.
func VisitDependencies(visitor DependencyVisitor, deps ...reflect.Value) {
	for _, dv := range deps {
		// for any structs that embed fx.In, recursively visit their fields
		if dig.IsIn(dv.Type()) {
			for stack := []reflect.Value{dv}; len(stack) > 0; {
				var (
					end           = len(stack) - 1
					container     = stack[end]
					containerType = container.Type()
				)

				stack = stack[:end]
				for i := 0; i < container.NumField(); i++ {
					field := containerType.Field(i)
					fieldValue := container.Field(i)

					// NOTE: skip unexported fields or those whose value cannot be accessed
					if len(field.PkgPath) > 0 ||
						!fieldValue.IsValid() ||
						!fieldValue.CanInterface() ||
						field.Type == InType() ||
						field.Type == OutType() {
						continue
					}

					if dig.IsIn(field.Type) {
						// this field is something that itself contains dependencies
						stack = append(stack, fieldValue)
					} else if !visitor(Dependency{Container: containerType, Field: &field, Value: fieldValue}) {
						return
					}
				}
			}
		} else if !visitor(Dependency{Value: dv}) { // a "naked" dependency
			return
		}
	}
}
