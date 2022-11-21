package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_serverConfiguration(t *testing.T) {
	expectedConfig := serverConfig{
		"127.0.0.1:8080",
		"",
		"./tmp/metricStorage.json",
		"key",
		0,
		true,
		true,
	}

	actualConfig := NewConfig(
		"127.0.0.1:8080",
		"",
		"./tmp/metricStorage.json",
		"key",
		0,
		true,
		true,
	)

	t.Run("config creation", func(t *testing.T) {
		assert.Equal(t, expectedConfig, actualConfig)
	})

	srv := NewServer(actualConfig)

	t.Run("server creation", func(t *testing.T) {
		assert.Equal(t, actualConfig, srv.config)
	})

	t.Run("server run without initialization", func(t *testing.T) {
		assert.ErrorIs(t, srv.Run(), errNotInitialized)
	})

	t.Run("server init with file storage", func(t *testing.T) {
		assert.NoError(t, srv.Init())
		assert.True(t, srv.initialized)
	})

	t.Run("server shutdown actions check", func(t *testing.T) {
		assert.NoError(t, srv.Shutdown())

		ok := true

		select {
		case _, ok = <-srv.uploadSig:
		default:
		}

		assert.False(t, ok)
	})

	srv.config.FileDestination = ""
	srv.config.DatabaseAddress = ""

	t.Run("server init with bad config", func(t *testing.T) {
		assert.ErrorIs(t, srv.Init(), errStorageNotDefined)
	})

}
