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

package arrange

import (
	"context"

	"go.uber.org/fx"
)

// Task is the type that any context-less operation must conform to.
type Task interface {
	~func() | ~func() error
}

// ShutdownWhenDone executes a context-less task and ensures that the enclosing fx.App
// is shutdown when the task is complete.  Any error, including a nil error, from the task
// is interpolated into an exit code via ExitCodeFor.  That exit code will be available
// in the fx.ShutdownSignal.
func ShutdownWhenDone[T Task](sh fx.Shutdowner, coder ErrorCoder, task T) (err error) {
	defer func() {
		sh.Shutdown(
			fx.ExitCode(
				ExitCodeFor(err, coder),
			),
		)
	}()

	if t, ok := any(task).(func()); ok {
		t()
	} else {
		err = any(task).(func() error)()
	}

	return
}

// TaskCtx is the type that any operation that requires a context must conform to.
type TaskCtx interface {
	~func(context.Context) | ~func(context.Context) error
}

// ShutdownWhenDoneCtx executes a context-based task and ensures that the enclosing fx.App
// is shutdown when the task is complete.  This function is like ShutdownWhenDone, but for use
// when a context.Context is needed.
func ShutdownWhenDoneCtx[T TaskCtx](ctx context.Context, sh fx.Shutdowner, coder ErrorCoder, task T) (err error) {
	defer func() {
		sh.Shutdown(
			fx.ExitCode(
				ExitCodeFor(err, coder),
			),
		)
	}()

	if t, ok := any(task).(func(context.Context)); ok {
		t(ctx)
	} else {
		err = any(task).(func(context.Context) error)(ctx)
	}

	return
}
