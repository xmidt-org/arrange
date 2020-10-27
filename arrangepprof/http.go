package arrangepprof

import (
	"net/http/pprof"
	"reflect"
	rpprof "runtime/pprof"

	"github.com/gorilla/mux"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
)

// DefaultPathPrefix is used as the path prefix for HTTP pprof handlers
// when no HTTPRoutes.PathPrefix field is supplied
const DefaultPathPrefix = "/debug/pprof"

// ConfigureRoutes adds the various pprof routes to a *mux.Router.  This
// function can be used as a standalone mechanism for adding pprof routes
// to any router.
//
// The typical way to use this function is to call it against a Subrouter, e.g.:
//
//   ConfigureRoutes(router.PathPrefix("/foo/").Subrouter())
//
// NOTE: This method does not do the special mapping for pprof.Index for
// a path prefix with no trailing slash, e.g. "/debug/pprof".  Callers will
// need to implement that.  HTTP.Provide handles that case.
func ConfigureRoutes(r *mux.Router) {
	r.Path("/").HandlerFunc(pprof.Index)
	r.Path("/cmdline").HandlerFunc(pprof.Cmdline)
	r.Path("/profile").HandlerFunc(pprof.Profile)
	r.Path("/symbol").HandlerFunc(pprof.Symbol)
	r.Path("/trace").HandlerFunc(pprof.Trace)

	// NOTE: have to add each profile separately, as gorilla/mux has
	// stricter matching and net/http.ServeMux
	for _, p := range rpprof.Profiles() {
		r.Path("/" + p.Name()).HandlerFunc(pprof.Index)
	}
}

// HTTP is the strategy for attaching pprof routes to an injected *mux.Router
type HTTP struct {
	// PathPrefix is the prefix URL for all the pprof routes.  If unset,
	// DefaultPathPrefix is used instead.  To bind the pprof routes to the
	// root URL, set this field to "/".
	PathPrefix string

	// RouterName is the fx.App component name of the *mux.Router.  If this field
	// is unset, then an unnamed, global *mux.Router is assumed.
	//
	// Set this field when multiple http.Server instances are used within
	// the same fx.App.
	RouterName string
}

// buildRouterIn dynamically builds an fx.In struct for Provide.
// The first two fields will the fx.In and the optional fx.Printer.
// The *mux.Router field will immediately follow those two.
func (hr *HTTP) buildRouterIn() reflect.Type {
	fields := []reflect.StructField{
		{
			Name:      "In",
			Type:      arrange.InType(),
			Anonymous: true,
		},
		{
			Name: "Printer",
			Type: arrange.PrinterType(),
			Tag:  `optional:"true"`,
		},
		{
			Name: "Router",
			Type: reflect.TypeOf((*mux.Router)(nil)),
		},
	}

	if len(hr.RouterName) > 0 {
		fields[1].Tag = reflect.StructTag(`name:"` + hr.RouterName + `"`)
	}

	return reflect.StructOf(fields)
}

// invokeFuncOf dynamically produces the function type for the fx.Invoke
// function that configures pprof routes
func (hr *HTTP) invokeFuncOf() reflect.Type {
	return reflect.FuncOf(
		// in
		[]reflect.Type{hr.buildRouterIn()},

		// out
		[]reflect.Type{},

		// not variadic
		false,
	)
}

// invoke is the implementation of the signature returned by invokeFuncOf
func (hr *HTTP) invoke(args []reflect.Value) []reflect.Value {
	var (
		in = args[0]

		p = arrange.NewModulePrinter(
			module,
			in.Field(1).Interface().(fx.Printer),
		)

		r = in.Field(2).Interface().(*mux.Router)

		prefix = hr.PathPrefix
	)

	if len(prefix) == 0 {
		prefix = DefaultPathPrefix
	}

	// strip trailing slashes to normalize the prefix
	for prefix[len(prefix)-1] == '/' {
		prefix = prefix[0 : len(prefix)-1]
	}

	p.Printf("Mapping pprof HTTP handlers to %s", prefix)

	// NOTE: this works if prefix was "/", as mapping to "" is valid
	r.HandleFunc(prefix, pprof.Index)
	ConfigureRoutes(r.PathPrefix(prefix + "/").Subrouter())
	return nil
}

// Provide returns an fx.Option that configures the net/http/pprof handlers
// for an injected *mux.Router
func (hr HTTP) Provide() fx.Option {
	return fx.Invoke(
		reflect.MakeFunc(
			hr.invokeFuncOf(),
			hr.invoke,
		).Interface(),
	)
}
