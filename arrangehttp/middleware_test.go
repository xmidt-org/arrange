/**
 * Copyright 2023 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package arrangehttp

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MiddlewareSuite struct {
	suite.Suite
}

func (suite *MiddlewareSuite) testApplyMiddleware(count int) {
	current := count - 1 // middleware themselves run in reverse order
	middleware := make([]func(http.Handler) http.Handler, 0, count)
	for i := 0; i < count; i++ {
		i := i
		middleware = append(middleware, func(actual http.Handler) http.Handler {
			suite.Same(http.DefaultServeMux, actual)
			suite.Equal(i, current)
			current--
			return actual
		})
	}

	suite.Equal(
		http.DefaultServeMux,
		ApplyMiddleware[http.Handler](http.DefaultServeMux, middleware...),
	)
}

func (suite *MiddlewareSuite) TestApplyMiddleware() {
	for _, count := range []int{0, 1, 2, 5} {
		suite.Run(fmt.Sprintf("count=%d", count), func() {
			suite.testApplyMiddleware(count)
		})
	}
}

func TestMiddleware(t *testing.T) {
	suite.Run(t, new(MiddlewareSuite))
}
