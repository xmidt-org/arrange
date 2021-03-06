package arrangepprof

import (
	"context"
	"errors"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
)

const module = "Arrange PPROF"

var (
	// ErrAlreadyProfiling indicates that a CPU profile to a particular path
	// has already been started
	ErrAlreadyProfiling = errors.New("CPU Profiling has already been started")
)

func openProfilePath(path string, overwrite bool) (*os.File, error) {
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !overwrite {
		flag |= os.O_EXCL
	}

	return os.OpenFile(path, flag, 0666)
}

// CPU is a strategy for managing CPU profiling within the lifecycle
// of an fx.App
type CPU struct {
	// Path is the optional file system path where the CPU profile data is written.
	// If this field is unset, i.e. the empty string, then no CPU profiling is done
	// by this object.
	Path string

	// Overwrite indicates whether the profile data stored in Path should be
	// overwritten when starting the fx.App.  By default, an error is raised if
	// Path already exists.
	Overwrite bool

	file *os.File
}

func (c *CPU) start() error {
	if len(c.Path) == 0 {
		return nil
	}

	if c.file != nil {
		return ErrAlreadyProfiling
	}

	var err error
	c.file, err = openProfilePath(c.Path, c.Overwrite)
	if err == nil {
		err = pprof.StartCPUProfile(c.file)
	}

	return err
}

func (c *CPU) stop() (err error) {
	if c.file != nil {
		pprof.StopCPUProfile()
		err = c.file.Close()
		c.file = nil
	}

	return
}

// Provide returns the necessary fx.App option to bind this CPU profiler
// to the application lifecycle.
func (c CPU) Provide() fx.Option {
	return fx.Invoke(
		func(in struct {
			fx.In
			Lifecycle fx.Lifecycle
			Printer   fx.Printer `optional:"true"`
		}) {
			// optimization: don't bother registering if Path is empty
			if len(c.Path) > 0 {
				p := arrange.NewModulePrinter(module, in.Printer)
				in.Lifecycle.Append(fx.Hook{
					OnStart: func(context.Context) error {
						p.Printf("Start CPU profiling to %s", c.Path)
						return c.start()
					},
					OnStop: func(context.Context) error {
						err := c.stop()
						p.Printf("Stop CPU profiling to %s", c.Path)
						return err
					},
				})
			}
		},
	)
}

// Heap is a strategy for writing memory profile data when
// an fx.App is stopped
type Heap struct {
	// Path is the optional file system path where the heap profile data is written.
	// If this field is unset, i.e. the empty string, then no heap profiling is done
	// by this object.
	Path string

	// Overwrite indicates whether the profile data stored in Path should be
	// overwritten when starting the fx.App.  By default, an error is raised if
	// Path already exists.
	Overwrite bool

	// DisableGCOnStop indicates whether runtime.GC is called when the fx.App stops.
	// By default, runtime.GC is called prior to writing heap profile data, as this gives
	// more accurate information.  Setting this will write heap profile data without
	// doing a runtime.GC.
	DisableGCOnStop bool
}

func (h *Heap) stop() (err error) {
	if len(h.Path) == 0 {
		return
	}

	var file *os.File
	file, err = openProfilePath(h.Path, h.Overwrite)
	if err == nil {
		if !h.DisableGCOnStop {
			runtime.GC()
		}

		err = pprof.WriteHeapProfile(file)
		file.Close()
	}

	return
}

// Provide returns the necessary fx.App option to bind this heap profiler
// to the application lifecycle.
func (h Heap) Provide() fx.Option {
	return fx.Invoke(
		func(in struct {
			fx.In
			Lifecycle fx.Lifecycle
			Printer   fx.Printer `optional:"true"`
		}) {
			// optimization: don't bother registering if Path is empty
			if len(h.Path) > 0 {
				p := arrange.NewModulePrinter(module, in.Printer)
				in.Lifecycle.Append(fx.Hook{
					OnStop: func(context.Context) error {
						err := h.stop()
						if err != nil {
							p.Printf("Error writing heap profile to %s: %s", h.Path, err)
						} else {
							p.Printf("Wrote heap profile data to %s", h.Path)
						}

						return err
					},
				})
			}
		},
	)
}
