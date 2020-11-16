package arrange

import (
	"reflect"
	"strconv"
	"strings"
)

// Field describes a single injected dependency as part of an
// enclosing struct.
type Field struct {
	// Name is the name of the component.  If set, Group is ignored.
	Name string

	// Group is the value group for this component.  Not used if Name is set.
	//
	// If this field is set, the actual type of the struct field will be a slice
	// with elements defined by the Type field.
	Group string

	// Optional indicates an optional component
	Optional bool

	// Type is the type for this component.  This may be anything TypeOf supports.
	//
	// If Group is set, the actual type of the generated struct field will be a slice
	// of this type.  Otherwise, the struct field will be of this type.
	Type interface{}
}

// Struct is a slice of reflect.StructFields with additional functionality
// that simplifies dynamically creating structs that participate in dependency
// injection.
type Struct []reflect.StructField

// In appends an anonymous (embedded) fx.In field.  No attempt is made
// to prevent this method being called multiple times.  Clients must ensure
// that multiple fx.In fields are not appended.
//
// This method returns the (possibly) extended Struct instance.
func (s Struct) In() Struct {
	return append(s, reflect.StructField{
		Name:      "In",
		Anonymous: true,
		Type:      InType(),
	})
}

// Append adds more dependencies to this sequence of fields.  Clients must
// ensure that duplicate fields are not appended.
//
// This method returns the (possibly) extended Struct instance.
func (s Struct) Append(more ...Field) Struct {
	var (
		b strings.Builder
		n []byte
	)

	for _, nf := range more {
		var sf reflect.StructField
		b.Reset()

		switch {
		case len(nf.Group) > 0:
			b.WriteString(`group:"`)
			b.WriteString(nf.Group)
			b.WriteString(`"`)
			sf.Type = reflect.SliceOf(TypeOf(nf.Type))

		case len(nf.Name) > 0:
			b.WriteString(`name:"`)
			b.WriteString(nf.Name)
			b.WriteString(`"`)
			fallthrough

		default:
			sf.Type = TypeOf(nf.Type)
		}

		if nf.Optional {
			if b.Len() > 0 {
				b.WriteRune(' ')
			}

			b.WriteString(`optional:"true"`)
		}

		sf.Tag = reflect.StructTag(b.String())
		b.Reset()
		b.WriteRune('F')
		n = strconv.AppendInt(n[:0], int64(len(s)), 10)
		b.Write(n)

		sf.Name = b.String()
		s = append(s, sf)
	}

	return s
}

// Extend appends the contents of several Struct instances and returns
// the (possibly) extended Struct instance.
func (s Struct) Extend(more ...Struct) Struct {
	for _, e := range more {
		s = append(s, e...)
	}

	return s
}

// Clone returns a distinct copy of this Struct instance.  Useful
// when a number of structs need to be generated that all have some
// common definitions.
func (s Struct) Clone() Struct {
	clone := make(Struct, len(s))
	copy(clone, s)
	return clone
}

// Of returns the struct type with the current set of fields
func (s Struct) Of() reflect.Type {
	return reflect.StructOf(s)
}
