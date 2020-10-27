package arrangepprof

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
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

func TestHTTP(t *testing.T) {
}
