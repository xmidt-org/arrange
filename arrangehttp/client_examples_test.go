package arrangehttp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
)

func ExampleClient_unmarshal() {
	const yaml = `
timeout: "45s"
`

	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(strings.NewReader(yaml))

	var client *http.Client
	app := fx.New(
		arrange.LoggerWriter(ioutil.Discard),
		arrange.ForViper(v),
		fx.Provide(
			func() RoundTripperConstructor {
				return func(next http.RoundTripper) http.RoundTripper {
					// you could add metrics, logging, whatever you want here
					return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
						request.Header.Set("Injected", "true")
						return next.RoundTrip(request)
					})
				}
			},
			Client().
				Inject(struct {
					fx.In

					// you can inject things from the enclosing fx.App
					M RoundTripperConstructor
				}{}).
				Middleware(
					func(next http.RoundTripper) http.RoundTripper {
						// you can also supply middleware from external sources
						return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
							request.Header.Set("External", "true")
							return next.RoundTrip(request)
						})
					},
				).
				With(func(c *http.Client) error {
					// you can tailor the client with options
					c.Timeout = 10 * time.Second // override!
					return nil
				}).
				Unmarshal(),
		),
		fx.Populate(&client),
	)

	if err := app.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't start app: %s", err)
		return
	}

	defer app.Stop(context.Background())
	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			fmt.Println("External", request.Header.Get("External"))
			fmt.Println("Injected", request.Header.Get("Injected"))
		}),
	)

	defer server.Close()

	request, err := http.NewRequest("GET", server.URL+"/example", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create request: %s", err)
		return
	}

	client.Do(request)

	// Output:
	// External true
	// Injected true
}

func ExampleClient_provideKey() {
	const yaml = `
clients:
  main:
    timeout: "45s"
`

	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(strings.NewReader(yaml))

	type ClientIn struct {
		fx.In
		Client *http.Client `name:"clients.main"` // notice that this is the same as our config key
	}

	var client *http.Client
	app := fx.New(
		arrange.LoggerWriter(ioutil.Discard),
		arrange.ForViper(v),
		Client().
			ProvideKey("clients.main"),
		fx.Invoke(
			func(in ClientIn) {
				client = in.Client
			},
		),
	)

	if err := app.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't start app: %s", err)
		return
	}

	defer app.Stop(context.Background())
	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(289) // just to verify the request got here
		}),
	)

	defer server.Close()

	request, err := http.NewRequest("GET", server.URL+"/example", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create request: %s", err)
		return
	}

	response, err := client.Do(request)
	if response != nil {
		fmt.Println(response.StatusCode)
	}

	// Output:
	// 289
}
