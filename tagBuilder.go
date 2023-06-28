package arrange

import (
	"reflect"
	"strings"

	"go.uber.org/fx"
)

// TagBuilder is a Fluent Builder for creating sequences of fx struct tags in various situations.
// This type enforces certain rules that fx requires, e.g. value groups
// cannot be optional, a component cannot be named and in a group, etc.
//
// Typical use is to start a chain of calls via the Tags function.  This yields
// safer alternative to simply declaring strings:
//
//	fx.New(
//	  fx.Provide(
//	    fx.Annotate(
//	      func(dep http.Handler) (*http.Server, error) { ... },
//	      arrange.Tags().Name("myHandler").ParamTags(),
//	      arrange.Tags().Name("myServer").ResultTags(),
//	    ),
//	  ),
//	)
type TagBuilder struct {
	prefixes []string
	tags     []string
}

// Push pushes a new prefix (or, scope) onto this builder.  Subsequent
// names and groups are prefixed with this string, separated by a '.'.
// For example:
//
//	arrange.Tags().Push("foo").Name("bar").ParamTags()
//
// results in a `name="foo.bar"` parameter tag.
//
// If the given prefix is the empty string, then no prefix is applied.
// This allows temporarily suspending prefixed names and groups during
// a sequence of tags by doing Push("") followed by Pop() and continuing
// with the previous prefix.
func (tb *TagBuilder) Push(prefix string) *TagBuilder {
	tb.prefixes = append(tb.prefixes, prefix)
	return tb
}

// Pop removes the most recent prefix established with Push.  If no
// prefixes are currently in use, this method does nothing.
func (tb *TagBuilder) Pop() *TagBuilder {
	if len(tb.prefixes) > 0 {
		tb.prefixes[len(tb.prefixes)-1] = ""
		tb.prefixes = tb.prefixes[0 : len(tb.prefixes)-1]
	}

	return tb
}

func (tb *TagBuilder) writePrefixedValue(o *strings.Builder, v string) {
	if len(tb.prefixes) > 0 {
		// allow prefixes to be blank, to "suspend" prefixing via Push("")
		if prefix := tb.prefixes[len(tb.prefixes)-1]; len(prefix) > 0 {
			o.WriteString(prefix)
			o.WriteRune('.')
		}
	}

	o.WriteString(v)
}

// Skip adds an empty tag to the sequence of tags under construction.
// Useful when a parameter or a result doesn't need any tag information, but
// there are subsequence parameters or results that do.
func (tb *TagBuilder) Skip() *TagBuilder {
	tb.tags = append(tb.tags, "")
	return tb
}

// Optional adds an `optional:"true"` tag to the sequence being built.
func (tb *TagBuilder) Optional() *TagBuilder {
	tb.tags = append(tb.tags, `optional:"true"`)
	return tb
}

// Name adds a `name:"..."` tag to the sequence being built.  Use OptionalName
// if a named component should also be optional.
func (tb *TagBuilder) Name(v string) *TagBuilder {
	var o strings.Builder
	o.WriteString(`name:"`)
	tb.writePrefixedValue(&o, v)
	o.WriteRune('"')
	tb.tags = append(tb.tags, o.String())

	return tb
}

// OptionalName adds a `name:"..." optional:"true"` tag to the sequence being built.
func (tb *TagBuilder) OptionalName(v string) *TagBuilder {
	var o strings.Builder
	o.WriteString(`name:"`)
	tb.writePrefixedValue(&o, v)
	o.WriteString(`" optional:"true"`)
	tb.tags = append(tb.tags, o.String())

	return tb
}

// Group adds a `group:"..."` tag to the sequence being built.  Groups cannot
// be optional.
func (tb *TagBuilder) Group(v string) *TagBuilder {
	var o strings.Builder
	o.WriteString(`group:"`)
	tb.writePrefixedValue(&o, v)
	o.WriteRune('"')
	tb.tags = append(tb.tags, o.String())

	return tb
}

// StructTags creates a sequence of reflect.StructTag objects using the
// previously described sequence of tags.
//
// This method does not reset the state of this builder.
func (tb *TagBuilder) StructTags() (tags []reflect.StructTag) {
	tags = make([]reflect.StructTag, 0, len(tb.tags))
	for _, v := range tb.tags {
		tags = append(tags, reflect.StructTag(v))
	}

	return
}

// ParamTags creates an fx.ParamTags annotation using the previously described
// sequence of tags.
//
// This method does not reset the state of this builder.
func (tb *TagBuilder) ParamTags() fx.Annotation {
	return fx.ParamTags(tb.tags...)
}

// ResultTags creates an fx.ResultTags annotation using the previously described
// sequence of tags.  Note that results cannot be marked as optional.  If one of the
// Optional methods of this builder was used to create a tag in the sequence, an error
// will short circuit fx.App startup.
//
// This method does not reset the state of this builder.
func (tb *TagBuilder) ResultTags() fx.Annotation {
	return fx.ResultTags(tb.tags...)
}

// Tags starts a Fluent Builder chain for creating a sequence of tags.
func Tags() *TagBuilder {
	return new(TagBuilder)
}
