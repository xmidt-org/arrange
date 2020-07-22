// Package arrange enhances uber/fx with some features commonly needed
// when using external configuration to drive the structure of the
// DI container.
//
// Unmarshaled components
//
// It's often useful to bootstrap one or more uber/fx components
// with state read from external configuration sources.  Arrange
// integrates with github.com/spf13/viper and supports unmarshaling
// components and injecting them into an fx.App.
//
// Consider the case of reading in some configuration.  Arrange allows
// you to easily unmarshal configuration into a struct, then use that struct
// as a dependency:
//
//   v := viper.New()
//   fx.New(
//     arrange.Supply(v), // you can use fx.Supply, but this has some extra features
//     arrange.Provide(Config{}), // this could also be a pointer
//     fx.Invoke(
//       func(cfg Config) error {
//         // use the Config as desired
//       },
//     ),
//   )
//
// Configuration keys are also supported:
//
//
//   type Components struct {
//     fx.In
//     Config Config `name:"server.main"`
//   }
//
//   v := viper.New()
//   fx.New(
//     arrange.Supply(v),
//     arrange.ProvideKey("server.main", Config{}), // this will be a named component
//     fx.Invoke(
//       func(c Components) error {
//         // use the Config as desired
//       },
//     ),
//   )
//
// Arrange also exposes a way to unmarshal several keys to the same type, which
// is commonly done when multiple components, such as http.Server objects, need to
// get bootstrapped:
//
//   type Components struct {
//     fx.In
//     Main    Config `name:"server.main"`
//     Health  Config `name:"server.health"`
//     Pprof   Config `name:"server.pprof"`
//     Control Config `name:"server.control"`
//   }
//
//   v := viper.New()
//   fx.New(
//     arrange.Supply(v),
//     arrange.Keys(
//       "server.main",
//       "server.health",
//       "server.pprof",
//       "server.control",
//     ).Unmarshal(Config{}), // these will be distinct, named instances of Config
//     fx.Invoke(
//       func(cfg Config) error {
//         // start all the servers
//       },
//     ),
//   )
//
// See the examples for additional features.
//
// Conditional options
//
// When creating an fx.App, it can be useful to conditionally include
// options based on external sources of information, such as the command-line,
// the environment, or even viper.  Arrange exposes a very simple API
// that makes conditional options simple and easy.
//
//   feature := flag.Bool("feature", false, "this is a feature flag")
//   flag.Parse()
//
//   app := fx.New(
//     arrange.If(os.Getenv("feature") != "" || feature).Then(
//       fx.Provide(
//         func() ConditionalComponent {
//           // this constructor only runs if the feature environment variable
//           // is set or if the feature command line flag was set
//         },
//       )
//     )
//   )
package arrange
