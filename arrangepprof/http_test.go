package arrangepprof

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestConfigureRoutes(t *testing.T) {
	router := new(mux.Router)
	ConfigureRoutes(router.PathPrefix("/foo/").Subrouter())

	// just spot check a few URLs, this isn't exhaustive
	testData := []string{
		"/foo/",
		"/foo/cmdline",
		"/foo/symbol",
		"/foo/trace",
		"/foo/allocs",
	}

	for _, url := range testData {
		t.Run(url, func(t *testing.T) {
			var (
				assert   = assert.New(t)
				response = httptest.NewRecorder()
				request  = httptest.NewRequest("GET", url, nil)
			)

			router.ServeHTTP(response, request)
			assert.Equal(http.StatusOK, response.Code)
			assert.Greater(response.Body.Len(), 0)
		})
	}
}

func testHTTPUnnamedRouter(t *testing.T) {
	var (
		assert = assert.New(t)
		router = new(mux.Router)
		app    = fxtest.New(
			t,
			fx.Supply(router),
			HTTP{}.Provide(),
		)
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/debug/pprof", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/debug/pprof/", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/debug/pprof/cmdline", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	app.RequireStop()
}

func testHTTPNamedRouter(t *testing.T) {
	var (
		assert = assert.New(t)
		router = new(mux.Router)
		app    = fxtest.New(
			t,
			fx.Provide(
				fx.Annotated{
					Name: "test",
					Target: func() *mux.Router {
						return router
					},
				},
			),
			HTTP{
				RouterName: "test",
			}.Provide(),
		)
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/debug/pprof", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/debug/pprof/", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/debug/pprof/cmdline", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	app.RequireStop()
}

func testHTTPCustomPathPrefix(t *testing.T) {
	var (
		assert = assert.New(t)
		router = new(mux.Router)
		app    = fxtest.New(
			t,
			fx.Supply(router),
			HTTP{
				PathPrefix: "/test/debug/",
			}.Provide(),
		)
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/test/debug", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/test/debug/", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/test/debug/cmdline", nil))
	assert.Equal(http.StatusOK, response.Code)
	assert.Greater(response.Body.Len(), 0)

	app.RequireStop()
}

func TestHTTP(t *testing.T) {
	t.Run("UnnamedRouter", testHTTPUnnamedRouter)
	t.Run("NamedRouter", testHTTPNamedRouter)
	t.Run("CustomPathPrefix", testHTTPCustomPathPrefix)
}
