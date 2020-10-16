// +build !linux
// +build !darwin
// +build !freebsd

package arrange

import (
	"fmt"

	"go.uber.org/fx"
)

// PluginSupported indicates whether plugins are supported on this platform.
// Useful to implement conditional components.
func PluginSupported() bool { return false }

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
	return fx.Error(
		fmt.Errorf("Unable to load plugin [%s]: plugins are not supported on this platform", path),
	)
}
