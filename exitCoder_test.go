package arrange

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testError struct{}

func (te testError) Error() string {
	return "test error"
}

type ExitCoderSuite struct {
	suite.Suite
}

func (suite *ExitCoderSuite) testUseExitCodeNilError() {
	suite.Panics(
		func() {
			UseExitCode(nil, 1)
		},
	)
}

func (suite *ExitCoderSuite) testUseExitCodeWithError() {
	err := UseExitCode(testError{}, 123)
	suite.Require().Error(err)

	var te testError
	suite.ErrorAs(err, &te)

	var ec ExitCoder
	suite.ErrorAs(err, &ec)
	suite.Equal(123, ec.ExitCode())
}

func (suite *ExitCoderSuite) TestUseExitCode() {
	suite.Run("NilError", suite.testUseExitCodeNilError)
	suite.Run("WithError", suite.testUseExitCodeWithError)
}

func (suite *ExitCoderSuite) testExitCodeForWithExitCoder() {
	suite.Run("NilErrorCoder", func() {
		err := UseExitCode(testError{}, 123)
		suite.Equal(123, ExitCodeFor(err, nil))
	})

	suite.Run("WithErrorCoder", func() {
		err := UseExitCode(testError{}, 123)
		suite.Equal(123, ExitCodeFor(err, func(error) int { return 255 }))
	})
}

func (suite *ExitCoderSuite) testExitCodeForNonExitCoder() {
	suite.Run("NilErrorCoder", func() {
		suite.Equal(
			DefaultErrorExitCode,
			ExitCodeFor(testError{}, nil),
		)
	})

	suite.Run("WithErrorCoder", func() {
		suite.Equal(
			255,
			ExitCodeFor(testError{}, func(error) int { return 255 }),
		)
	})
}

func (suite *ExitCoderSuite) testExitCodeForNilError() {
	suite.Run("NilErrorCoder", func() {
		suite.Equal(0, ExitCodeFor(nil, nil))
	})

	suite.Run("WithErrorCoder", func() {
		suite.Equal(
			255,
			ExitCodeFor(nil, func(v error) int {
				suite.NoError(v)
				return 255
			}),
		)
	})
}

func (suite *ExitCoderSuite) TestExitCodeFor() {
	suite.Run("WithExitCoder", suite.testExitCodeForWithExitCoder)
	suite.Run("NonExitCoder", suite.testExitCodeForNonExitCoder)
	suite.Run("NilError", suite.testExitCodeForNilError)
}

func TestExitCoder(t *testing.T) {
	suite.Run(t, new(ExitCoderSuite))
}
