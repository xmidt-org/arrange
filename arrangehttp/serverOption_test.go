package arrangehttp

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange"
)

func TestBaseContext(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		server      = new(http.Server)
		expectedCtx = context.WithValue(context.Background(), "test", "yes")
	)

	require.NoError(
		arrange.Invoke{
			BaseContext(func(net.Listener) context.Context {
				return expectedCtx
			}),
		}.Apply(server),
	)

	require.NotNil(server.BaseContext)
	assert.Equal(
		expectedCtx,
		server.BaseContext(nil),
	)
}

func TestConnContext(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		server  = new(http.Server)
		baseCtx = context.WithValue(context.Background(), "base", "yes")
		connCtx = context.WithValue(baseCtx, "conn", "yes")
	)

	require.NoError(
		arrange.Invoke{
			ConnContext(func(ctx context.Context, _ net.Conn) context.Context {
				assert.Equal(baseCtx, ctx)
				return connCtx
			}),
		}.Apply(server),
	)

	require.NotNil(server.ConnContext)
	assert.Equal(
		connCtx,
		server.ConnContext(baseCtx, nil),
	)
}

func TestErrorLog(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		server  = new(http.Server)

		output   bytes.Buffer
		errorLog = log.New(&output, "test", log.LstdFlags)
	)

	require.NoError(
		arrange.Invoke{
			ErrorLog(errorLog),
		}.Apply(server),
	)

	require.NotNil(server.ErrorLog)
	server.ErrorLog.Printf("an error")
	assert.NotEmpty(output.String())
}
