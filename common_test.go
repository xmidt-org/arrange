package arrange

import (
	"testing"
	"time"
)

// testPrinter is used to redirect fx.App logging to the testing.T object.
// This prevents spamminess when -v is not set.
type testPrinter struct{ *testing.T }

func (tp testPrinter) Printf(msg string, args ...interface{}) {
	tp.T.Logf(msg, args...)
}

type TestConfig struct {
	Name     string
	Age      int
	Interval time.Duration
}

// AnotherConfig is a type alias that prevent collisions when multiple
// TestConfigs need to be read from viper.
type AnotherConfig TestConfig
