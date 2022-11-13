package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_serverConfiguration(t *testing.T) {
	expectedConfig := serverConfig{
		"127.0.0.1:8080",
		"",
		"./tmp/metricStorage.json",
		"key",
		time.Second,
		true,
		false,
	}

	actualConfig := NewConfig(
		"127.0.0.1:8080",
		"",
		"./tmp/metricStorage.json",
		"key",
		time.Second,
		true,
		true,
	)

	t.Run("config creation", func(t *testing.T) {
		assert.Equal(t, expectedConfig, actualConfig)
		assert.NoError(t, actualConfig.SetByExternal())
	})

	srv := NewServer(actualConfig)

	t.Run("server creation", func(t *testing.T) {
		assert.Equal(t, actualConfig, srv.config)
	})

	t.Run("server run without initialization", func(t *testing.T) {
		assert.ErrorIs(t, srv.Run(), errNotInitialized)
	})

	t.Run("server init with good config", func(t *testing.T) {
		assert.NoError(t, srv.Init())
	})

	srv.config.FileDestination = ""
	srv.config.DatabaseAddress = ""

	t.Run("server init with bad config", func(t *testing.T) {
		assert.ErrorIs(t, srv.Init(), errStorageNotDefined)
	})

	t.Run("server shutdown actions check", func(t *testing.T) {
		assert.NoError(t, srv.Shutdown())

		ok := true

		select {
		case _, ok = <-srv.syncUpload:
		default:
		}

		assert.False(t, ok, errStorageNotDefined)
	})
}
