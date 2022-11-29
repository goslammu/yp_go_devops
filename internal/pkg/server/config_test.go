package server

import (
	"testing"

	log "github.com/sirupsen/logrus"
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
	})

	srv.config.FileDestination = "./tmp/metricStorage.json"

	t.Run("server init with file storage", func(t *testing.T) {
		assert.NoError(t, srv.Init())
		assert.True(t, srv.initialized)
	})

	srv.turnedOn = true

	t.Run("server shutdown actions check", func(t *testing.T) {
		assert.NoError(t, srv.Shutdown())

		ok := true

		select {
		case _, ok = <-srv.uploadSig:
		default:
		}
		log.Println("---------- UPSIG ", srv.uploadSig == nil)
		assert.False(t, ok)
	})
}
