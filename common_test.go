package arrange

import (
	"errors"
	"fmt"
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

// badWriter is an io.Writer that always returns an error
type badWriter struct{}

func (ae badWriter) Write([]byte) (int, error) {
	return 0, errors.New("expected io.Writer error")
}

type badUnmarshaler struct{}

func (bu badUnmarshaler) Unmarshal(interface{}) error {
	return errors.New("expected Unmarshal error")
}

func (bu badUnmarshaler) UnmarshalKey(key string, _ interface{}) error {
	return fmt.Errorf("expected UnmarshalKey error from [%s]", key)
}
