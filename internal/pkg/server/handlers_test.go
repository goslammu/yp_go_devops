package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dcaiman/YP_GO/internal/pkg/filestorage"
	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/stretchr/testify/assert"
)

func Test_handlerGetMetric(t *testing.T) {
	srv := server{}

	t.Run("unknown type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/value/unknownType/name", nil)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerGetMetric)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusNotImplemented, status)
	})
}

func Test_handlerGetMetricJSON(t *testing.T) {
	srv := server{}

	t.Run("empty json", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/value", bytes.NewBuffer(nil))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerGetMetricJSON)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("bad json", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/value", bytes.NewBuffer([]byte(
			`
			[{
				"unknownField": "unknownType"
			}]
			`)))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerGetMetricJSON)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("unsupported type", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/value", bytes.NewBuffer([]byte(
			`
			{
				"type": "unknownType"
			}
			`)))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerGetMetricJSON)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusNotImplemented, status)
	})
}

func Test_handlerUpdateBatch(t *testing.T) {
	srv := server{}

	t.Run("empty batch json", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/updates", bytes.NewBuffer(nil))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerUpdateBatch)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("bad batch json", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/updates", bytes.NewBuffer([]byte(
			`
			{
				"type": "unknownType"
			}
			`)))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerUpdateBatch)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("empty batch hash", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/updates", bytes.NewBuffer([]byte(
			`[
				{
					"id":"metric",
					"type":"Gauge"
				}
			]`)))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerUpdateBatch)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("bad batch hash", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/updates", bytes.NewBuffer([]byte(
			`[
				{
					"id":"metric",
					"type":"Gauge",
					"hash": "111"
				}
			]`)))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerUpdateBatch)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusBadRequest, status)
	})
}

func Test_handlerCheckConnection(t *testing.T) {
	srv := server{
		storage: filestorage.New(""),
	}

	t.Run("filestorage", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/ping", nil)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(srv.handlerCheckConnection)

		handler.ServeHTTP(rec, req)

		status := rec.Code
		assert.Equal(t, http.StatusOK, status)
	})
}

func Test_checkTypeSupport(t *testing.T) {
	tests := []struct {
		ExpectedError error
		Name          string
		Input         string
	}{
		{
			Name:          "gauge support",
			Input:         Gauge,
			ExpectedError: nil,
		},
		{
			Name:          "counter support",
			Input:         Counter,
			ExpectedError: nil,
		},
		{
			Name:          "unknown type",
			Input:         "type",
			ExpectedError: errUnsupportedType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			assert.ErrorIs(t, checkTypeSupport(tt.Input), tt.ExpectedError)
		})
	}
}

func Test_checkHash(t *testing.T) {
	srv := server{
		config: serverConfig{
			HashKey: "key",
		},
	}

	var delta int64 = 100
	var value float64 = 200
	mBadHash := &metric.Metric{
		ID:    "id",
		MType: "type",
		Value: &value,
		Delta: &delta,
		Hash:  "111",
	}
	mEmptyHash := &metric.Metric{
		ID:    "id",
		MType: "type",
		Value: &value,
		Delta: &delta,
	}
	mGoodHash := &metric.Metric{
		ID:    "id",
		MType: "type",
		Value: &value,
		Delta: &delta,
	}
	err := mGoodHash.UpdateHash(srv.config.HashKey)
	assert.NoError(t, err)

	tests := []struct {
		Input         *metric.Metric
		ExpectedError error
		Name          string
		ExpectedHash  string
	}{
		{
			Name:          "good hash",
			Input:         mGoodHash,
			ExpectedHash:  mGoodHash.Hash,
			ExpectedError: nil,
		},
		{
			Name:          "bad hash",
			Input:         mBadHash,
			ExpectedHash:  "",
			ExpectedError: errInconsistentHashes,
		},
		{
			Name:          "empty hash",
			Input:         mEmptyHash,
			ExpectedHash:  "",
			ExpectedError: errInconsistentHashes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			hash, err := srv.checkHash(tt.Input)

			assert.Equal(t, tt.ExpectedHash, hash)
			assert.ErrorIs(t, err, tt.ExpectedError)
		})
	}
}
