package arrange

import (
	"time"
)

type TestConfig struct {
	Name     string
	Age      int
	Interval time.Duration
}

// AnotherConfig is a type alias that prevent collisions when multiple
// TestConfigs need to be read from viper.
type AnotherConfig TestConfig
