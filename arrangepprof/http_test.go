// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrangepprof

import (
	"net/http"
	"net/url"
	"path"
	"testing"

	"github.com/stretchr/testify/suite"
)

type HTTPSuite struct {
	suite.Suite
}

// assertPprofRoutes verifies that the multiplexer was correctly configured
func (suite *HTTPSuite) assertPprofRoutes(mux *http.ServeMux, expectedPathPrefix string) {
	suite.Require().NotNil(mux)

	suite.HTTPSuccess(mux.ServeHTTP, http.MethodGet, expectedPathPrefix, nil)
	suite.HTTPSuccess(mux.ServeHTTP, http.MethodGet, expectedPathPrefix+"/", nil)
	suite.HTTPSuccess(mux.ServeHTTP, http.MethodGet, path.Join(expectedPathPrefix, "/cmdline"), nil)

	// the profile endpoint will block for 30s by default, which we don't want in a unit test
	profileQuery, err := url.ParseQuery("seconds=1")
	suite.Require().NoError(err)
	suite.HTTPSuccess(mux.ServeHTTP, "GET", path.Join(expectedPathPrefix, "/profile"), profileQuery)

	suite.HTTPSuccess(mux.ServeHTTP, http.MethodGet, path.Join(expectedPathPrefix, "/symbol"), nil)
	suite.HTTPSuccess(mux.ServeHTTP, http.MethodGet, path.Join(expectedPathPrefix, "/trace"), nil)
}

func (suite *HTTPSuite) testApply(expectedPathPrefix, configuredPathPrefix string) {
	mux := HTTP{
		PathPrefix: configuredPathPrefix,
	}.Apply(http.NewServeMux())

	suite.assertPprofRoutes(mux, expectedPathPrefix)
}

func (suite *HTTPSuite) TestApply() {
	suite.Run("DefaultPathPrefix", func() {
		suite.testApply(DefaultPathPrefix, "")
	})

	suite.Run("CustomPathPrefix", func() {
		suite.testApply("/custom", "/custom")
	})
}

func (suite *HTTPSuite) testNew(expectedPathPrefix, configuredPathPrefix string) {
	mux := HTTP{
		PathPrefix: configuredPathPrefix,
	}.New()

	suite.assertPprofRoutes(mux, expectedPathPrefix)
}

func (suite *HTTPSuite) TestNew() {
	suite.Run("DefaultPathPrefix", func() {
		suite.testNew(DefaultPathPrefix, "")
	})

	suite.Run("CustomPathPrefix", func() {
		suite.testNew("/custom", "/custom")
	})
}

func TestHTTP(t *testing.T) {
	suite.Run(t, new(HTTPSuite))
}
