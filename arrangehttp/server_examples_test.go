package arrangehttp

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
)

func ExampleServer_unmarshal() {
	const yaml = `
address: ":0"
`

	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(strings.NewReader(yaml))

	type Handlers struct {
		fx.In

		Api    http.Handler `name:"api"`
		Health http.Handler `name:"health"`
	}

	address := make(chan net.Addr, 1)
	app := fx.New(
		arrange.LoggerWriter(ioutil.Discard),
		arrange.ForViper(v),
		fx.Provide(
			fx.Annotated{
				Name: "api",
				Target: func() http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Write([]byte("for make glorious API!\n"))
					})
				},
			},
			fx.Annotated{
				Name: "health",
				Target: func() http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Write([]byte("so very healthy!\n"))
					})
				},
			},
			func() mux.MiddlewareFunc {
				return func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						fmt.Println("Hello from Middlewaretopia!")
						next.ServeHTTP(response, request)
					})
				}
			},
			Server().
				CaptureListenAddress(address).
				Inject(struct {
					fx.In
					M mux.MiddlewareFunc
				}{}).
				Unmarshal(),
		),
		fx.Invoke(
			func(r *mux.Router, h Handlers) {
				r.Handle("/api", h.Api)
				r.Handle("/health", h.Health)
			},
		),
	)

	if err := app.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't start app: %s", err)
		return
	}

	defer app.Stop(context.Background())
	serverURL := "http://" + MustGetListenAddress(address, time.After(time.Second)).String()
	if response, err := http.Get(serverURL + "/api"); err == nil {
		io.Copy(os.Stdout, response.Body)
		response.Body.Close()
	}

	if response, err := http.Get(serverURL + "/health"); err == nil {
		io.Copy(os.Stdout, response.Body)
		response.Body.Close()
	}

	// Output:
	// Hello from Middlewaretopia!
	// for make glorious API!
	// Hello from Middlewaretopia!
	// so very healthy!
}

func ExampleServer_provideKey() {
	const yaml = `
servers:
  main:
    address: ":0"
	readTimeout: "30s"
`

	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(strings.NewReader(yaml))

	type RouterIn struct {
		fx.In

		Router  *mux.Router `name:"servers.main"` // notice that this is the same as our config key
		Handler http.Handler
	}

	address := make(chan net.Addr, 1)
	app := fx.New(
		arrange.LoggerWriter(ioutil.Discard),
		arrange.ForViper(v),
		fx.Provide(
			func() http.Handler {
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					response.Write([]byte("Baking the API cookies\n"))
				})
			},
			fx.Annotated{
				Name: "first",
				Target: func() mux.MiddlewareFunc {
					return func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
							fmt.Println("first middleware")
							next.ServeHTTP(response, request)
						})
					}
				},
			},
			fx.Annotated{
				Name: "second",
				Target: func() mux.MiddlewareFunc {
					return func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
							fmt.Println("second middleware")
							next.ServeHTTP(response, request)
						})
					}
				},
			},
		),
		// this is outside fx.Provide(...)
		Server().
			Inject(struct {
				fx.In

				// NOTE: the order in which middleware is applied is the
				// same as the declared order in this struct
				M1 mux.MiddlewareFunc `name:"first"`
				M2 mux.MiddlewareFunc `name:"second"`
			}{}).
			CaptureListenAddress(address).
			ProvideKey("servers.main"),
		fx.Invoke(
			func(in RouterIn) {
				in.Router.Handle("/api", in.Handler)
			},
		),
	)

	if err := app.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't start app: %s", err)
		return
	}

	defer app.Stop(context.Background())
	serverURL := "http://" + MustGetListenAddress(address, time.After(time.Second)).String()
	if response, err := http.Get(serverURL + "/api"); err == nil {
		io.Copy(os.Stdout, response.Body)
		response.Body.Close()
	}

	// Output:
	// first middleware
	// second middleware
	// Baking the API cookies
}
