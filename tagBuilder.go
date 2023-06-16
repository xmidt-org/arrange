package arrange

import (
	"reflect"
	"strings"

	"go.uber.org/fx"
)

type TagBuilder struct {
	tags []string
}

func (tb *TagBuilder) Skip() *TagBuilder {
	tb.tags = append(tb.tags, "")
	return tb
}

func (tb *TagBuilder) Optional() *TagBuilder {
	tb.tags = append(tb.tags, `optional:"true"`)
	return tb
}

func (tb *TagBuilder) Name(v string) *TagBuilder {
	var o strings.Builder
	o.WriteString(`name:"`)
	o.WriteString(v)
	o.WriteRune('"')
	tb.tags = append(tb.tags, o.String())

	return tb
}

func (tb *TagBuilder) OptionalName(v string) *TagBuilder {
	var o strings.Builder
	o.WriteString(`name:"`)
	o.WriteString(v)
	o.WriteString(`" optional:"true"`)
	tb.tags = append(tb.tags, o.String())

	return tb
}

func (tb *TagBuilder) Group(v string) *TagBuilder {
	var o strings.Builder
	o.WriteString(`group:"`)
	o.WriteString(v)
	o.WriteRune('"')
	tb.tags = append(tb.tags, o.String())

	return tb
}

func (tb *TagBuilder) StructTags() (tags []reflect.StructTag) {
	tags = make([]reflect.StructTag, 0, len(tb.tags))
	for _, v := range tb.tags {
		tags = append(tags, reflect.StructTag(v))
	}

	return
}

func (tb *TagBuilder) ParamTags() fx.Annotation {
	return fx.ParamTags(tb.tags...)
}

func (tb *TagBuilder) ResultTags() fx.Annotation {
	return fx.ResultTags(tb.tags...)
}

func Tags() *TagBuilder {
	return new(TagBuilder)
}
