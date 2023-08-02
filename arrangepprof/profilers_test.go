// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrangepprof

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
)

func testCPUProvideDisabled(t *testing.T) {
	assert := assert.New(t)
	app := fxtest.New(
		t,
		CPU{}.Provide(), // no Path means CPU profiling should be disabled
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	assert.NoError(app.Err())
	app.RequireStop()
}

func testCPUStartDisabled(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		cpu     CPU
	)

	require.NoError(cpu.start())

	// no Path means start should be a nop
	assert.Nil(cpu.file)
	assert.NoError(cpu.stop())
	assert.NoError(cpu.stop()) // idempotent
}

func testCPUAlreadyProfiling(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		path    = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("testCPUAlreadyProfiling-%d.prof", rand.Int()), //nolint:gosec
		)

		cpu = CPU{
			Path: path,
		}
	)

	require.NoError(cpu.start())
	assert.Error(cpu.start())
	assert.NoError(cpu.stop())
	assert.NoError(cpu.stop()) // idempotent
}

func testCPUNewPath(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		path    = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("testCPUNewPath-%d.prof", rand.Int()), //nolint:gosec
		)

		app = fxtest.New(
			t,
			CPU{
				Path: path,
			}.Provide(),
		)
	)

	defer os.Remove(path)

	app.RequireStart()
	defer app.Stop(context.Background())
	assert.NoError(app.Err())
	app.RequireStop()

	info, err := os.Stat(path)
	require.NoError(err)
	assert.Greater(info.Size(), int64(0))
}

func testCPUExistingPath(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		path    = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("testCPUExistingPath-%d.prof", rand.Int()), //nolint:gosec
		)
	)

	f, err := os.Create(path)
	require.NoError(err)
	require.NoError(f.Close())
	defer os.Remove(path)

	app := fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return fxtest.NewTestLogger(t)
		}),
		CPU{
			Path: path,
		}.Provide(),
	)

	assert.Error(
		app.Start(context.Background()),
	)

	info, err := os.Stat(path)
	require.NoError(err)
	assert.Zero(info.Size())
}

func testCPUOverwrite(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		path    = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("testCPUOverwrite-%d.prof", rand.Int()), //nolint:gosec
		)
	)

	f, err := os.Create(path)
	require.NoError(err)
	require.NoError(f.Close())
	defer os.Remove(path)

	app := fxtest.New(
		t,
		CPU{
			Path:      path,
			Overwrite: true,
		}.Provide(),
	)

	app.RequireStart()
	defer app.Stop(context.Background())
	assert.FileExists(path)
	app.RequireStop()

	info, err := os.Stat(path)
	require.NoError(err)
	assert.Greater(info.Size(), int64(0))
}

func TestCPU(t *testing.T) {
	t.Run("ProvideDisabled", testCPUProvideDisabled)
	t.Run("StartDisabled", testCPUStartDisabled)
	t.Run("AlreadyProfiling", testCPUAlreadyProfiling)
	t.Run("NewPath", testCPUNewPath)
	t.Run("ExistingPath", testCPUExistingPath)
	t.Run("Overwrite", testCPUOverwrite)
}

func testHeapProvideDisabled(t *testing.T) {
	assert := assert.New(t)
	app := fxtest.New(
		t,
		Heap{}.Provide(), // no Path means heap profiling should be disabled
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	assert.NoError(app.Err())
	app.RequireStop()
}

func testHeapStopDisabled(t *testing.T) {
	var (
		assert = assert.New(t)
		heap   Heap
	)

	// no Path means start should be a nop
	assert.NoError(heap.stop())
	assert.NoError(heap.stop()) // idempotent
}

func testHeapNewPath(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		path    = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("testHeapNewPath-%d.prof", rand.Int()), //nolint:gosec
		)

		app = fxtest.New(
			t,
			Heap{
				Path: path,
			}.Provide(),
		)
	)

	defer os.Remove(path)

	app.RequireStart()
	defer app.Stop(context.Background())

	assert.NoError(app.Err())
	app.RequireStop()

	info, err := os.Stat(path)
	require.NoError(err)
	assert.Greater(info.Size(), int64(0))
}

func testHeapExistingPath(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		path    = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("testHeapExistingPath-%d.prof", rand.Int()), //nolint:gosec
		)
	)

	f, err := os.Create(path)
	require.NoError(err)
	require.NoError(f.Close())
	defer os.Remove(path)

	app := fxtest.New(
		t,
		Heap{
			Path: path,
		}.Provide(),
	)

	require.NoError(
		app.Start(context.Background()),
	)

	assert.Error(
		app.Stop(context.Background()),
	)

	info, err := os.Stat(path)
	require.NoError(err)
	assert.Zero(info.Size())
}

func testHeapOverwrite(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		path    = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("testHeapOverwrite-%d.prof", rand.Int()), //nolint:gosec
		)
	)

	f, err := os.Create(path)
	require.NoError(err)
	require.NoError(f.Close())
	defer os.Remove(path)

	app := fxtest.New(
		t,
		Heap{
			Path:      path,
			Overwrite: true,
		}.Provide(),
	)

	app.RequireStart()
	defer app.Stop(context.Background())
	assert.FileExists(path)
	app.RequireStop()

	assert.FileExists(path)
	info, err := os.Stat(path)
	require.NoError(err)
	assert.Greater(info.Size(), int64(0))
}

func TestHeap(t *testing.T) {
	t.Run("ProvideDisabled", testHeapProvideDisabled)
	t.Run("StopDisabled", testHeapStopDisabled)
	t.Run("NewPath", testHeapNewPath)
	t.Run("ExistingPath", testHeapExistingPath)
	t.Run("HeapOverwrite", testHeapOverwrite)
}
