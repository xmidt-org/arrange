package arrange

import "go.uber.org/fx"

// Conditional is a simple strategy for emitting options into
// an fx.App container
type Conditional struct {
}

// Then returns all the given options if this Conditional is not nil.
// If this Conditional is nil, it returns an empty fx.Options.
func (c *Conditional) Then(o ...fx.Option) fx.Option {
	if c != nil {
		return fx.Options(o...)
	}

	return fx.Options()
}

// If returns a non-nil Conditional if its sole argument is true.
// This allows one to build up conditional components:
//
//	v := viper.New() // initialize
//	fx.New(
//	  fx.Supply(v),
//
//	  // it's safe to provide this unconditionally as fx will not invoke
//	  // this constructor unless needed
//	  arrange.ProvideKey("server.main", ServerConfig{}),
//
//	  arrange.If(v.IsSet("server.main")).Then(
//	    fx.Invoke(
//	      func(cfg ServerConfig) error {
//	        // use the configuration to start the server
//	      },
//	    ),
//	  ),
//
//	  arrange.IfNot(v.IsSet("server.main")).Then(
//	    fx.Invoke(
//	      func() {
//	        log.Println("Main server not started")
//	      },
//	    ),
//	  ),
//	)
//
// Note that conditional components do not have to use viper.  Any function or series
// of boolean operators may be used:
//
//	feature := flag.Bool("feature", false, "this is a feature flag")
//	flag.Parse()
//	fx.New(
//	  arrange.If(os.Getenv("feature") != "" || feature).Then(
//	    fx.Provide(
//	      func() ConditionalComponent {
//	        return ConditionalComponent{}
//	      },
//	    ),
//	    fx.Invoke(
//	      func(cc ConditionalComponent) error {
//	        // start whatever is needed for this conditionally enabled component
//	      },
//	    ),
//	  )
//	)
func If(f bool) *Conditional {
	if f {
		return new(Conditional)
	}

	return nil
}

// IfNot is the boolean inverse of If
func IfNot(f bool) *Conditional {
	if !f {
		return new(Conditional)
	}

	return nil
}
