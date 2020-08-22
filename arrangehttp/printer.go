package arrangehttp

import "github.com/xmidt-org/arrange"

func prepend(template string) string {
	return arrange.Prepend("Arrange HTTP", template)
}
