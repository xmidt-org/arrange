// +build linux darwin freebsd

package arrange

import (
	"fmt"
	"plugin"
	"reflect"

	"go.uber.org/fx"
)

// PluginSupported indicates whether plugins are supported on this platform.
// Useful to implement conditional components.
func PluginSupported() bool { return true }

// ProvidePlugin loads a plugin's symbols and makes them available to the
// enclosing fx.App.
//
// Function symbols are passed to fx.Provide.  Plugins can leverage this to
// integrate with the fx.App directly.
//
// Nonfunctions are passed to fx.Supply and are thus available as global components.
//
// Any errors in loading either the plugin or any of its symbols will shortcircuit
// application startup.
func ProvidePlugin(path string, symbols ...string) fx.Option {
	p, err := plugin.Open(path)
	if err != nil {
		return fx.Error(
			fmt.Errorf("Unable to load plugin [%s]: %s", path, err),
		)
	}

	var options []fx.Option
	for _, symName := range symbols {
		s, err := p.Lookup(symName)
		if err != nil {
			// NOTE: even if there are symbol loading errors, we want to attempt
			// to load everything to give a complete picture of any issues
			options = append(options, fx.Error(err))
			continue
		}

		t := reflect.TypeOf(s)
		if t.Kind() == reflect.Func {
			options = append(options, fx.Provide(s))
		} else {
			options = append(options, fx.Supply(s))
		}
	}

	return fx.Options(options...)
}
