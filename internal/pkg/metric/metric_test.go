package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_UpdateHash(t *testing.T) {
	var delta1 int64 = 100
	var delta2 int64 = 0
	var value1 float64 = 200
	var value2 float64 = 0

	tests := []struct {
		Name         string
		Metric       *Metric
		ExpectedHash string
		Key          string
	}{
		{
			Name: "simple metric",
			Metric: &Metric{
				ID:    "id",
				MType: "type",
				Delta: &delta1,
				Value: &value1,
			},
			ExpectedHash: "72b53cfe218fd3f4d8fd080e41706f8d7bd2eeb3e0719f8788c857d9639052c8",
			Key:          "key1",
		},
		{
			Name: "zero delta",
			Metric: &Metric{
				ID:    "id",
				MType: "type",
				Delta: &delta2,
				Value: &value1,
			},
			ExpectedHash: "396b0d235a4790a2684cdddaf01f21ffe130de95c70b4f809ecc0ce45ccd01c9",
			Key:          "key2",
		},
		{
			Name: "zero value",
			Metric: &Metric{
				ID:    "id",
				MType: "type",
				Delta: &delta1,
				Value: &value2,
			},
			ExpectedHash: "ba5db309e6bc23446f4c4fe1c08e4e8f28a1a3f8e3218b46d28bb169a899c960",
			Key:          "key3",
		},
		{
			Name: "empty key",
			Metric: &Metric{
				ID:    "id",
				MType: "type",
				Delta: &delta1,
				Value: &value2,
			},
			ExpectedHash: "",
			Key:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			assert.NoError(t, tt.Metric.UpdateHash(tt.Key))
			assert.Equal(t, tt.ExpectedHash, tt.Metric.Hash)
		})
	}
}
