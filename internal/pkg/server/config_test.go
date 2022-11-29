package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_serverConfiguration(t *testing.T) {
	expectedConfig := serverConfig{
		"127.0.0.1:8080",
		"",
		"",
		"key",
		"privateCryptoKey",
		0,
		InitialDownloadOn,
		DropDatabaseOn,
		ModeHTTP,
	}

	actualConfig := NewConfig(
		"127.0.0.1:8080",
		"",
		"",
		"key",
		"privateCryptoKey",
		0,
		InitialDownloadOn,
		DropDatabaseOn,
		ModeHTTP,
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

	t.Run("server init with bad config", func(t *testing.T) {
		assert.ErrorIs(t, srv.Init(), errStorageNotDefined)
		assert.False(t, srv.initialized)
	})

	srv.config.FileDestination = "./tmp/metricStorage.json"

	t.Run("server init with file storage and shutdown", func(t *testing.T) {
		assert.NoError(t, srv.Init())
		assert.True(t, srv.initialized)

		srv.turnedOn = true

		ok := false

		select {
		case _, ok = <-srv.uploadSig:
		default:
			ok = true
		}

		assert.True(t, ok)

		assert.NoError(t, srv.Shutdown())

		select {
		case _, ok = <-srv.uploadSig:
		default:
		}

		assert.False(t, ok)

	})
}
