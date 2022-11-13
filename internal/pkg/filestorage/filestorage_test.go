package filestorage

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	filePath := "filepath"
	ms := New(filePath)

	assert.Equal(t, ms.FilePath, filePath)
	assert.NotNil(t, ms.metrics)
}

func Test_GetMetric(t *testing.T) {
	var delta int64 = 100
	var value float64 = 200
	m := metric.Metric{
		ID:    "name",
		MType: "type",
		Delta: &delta,
		Value: &value,
	}

	ms := New("")
	ms.metrics[m.ID] = &m

	tests := []struct {
		ExpectedMetric *metric.Metric
		ExpectedError  error
		Name           string
		MetricID       string
	}{
		{
			Name:           "existing metric",
			MetricID:       "name",
			ExpectedMetric: &m,
			ExpectedError:  nil,
		},
		{
			Name:           "non-existing metric",
			MetricID:       "otherName",
			ExpectedMetric: nil,
			ExpectedError:  metric.ErrMetricDoesntExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			m, err := ms.GetMetric(tt.MetricID)
			assert.ErrorIs(t, err, tt.ExpectedError)

			assert.Equal(t, tt.ExpectedMetric, m)
		})
	}
}

func Test_GetBatch(t *testing.T) {
	ms := New("")
	expectedMetrics := []*metric.Metric{}

	for i := 0; i < 10; i++ {
		delta := int64(3 * i)
		value := float64(4 * i)
		m := metric.Metric{
			ID:    "metric" + fmt.Sprint(i),
			MType: "type" + fmt.Sprint(2*i),
			Delta: &delta,
			Value: &value,
		}

		expectedMetrics = append(expectedMetrics, &m)
		ms.metrics[m.ID] = &m
	}
	sort.Slice(expectedMetrics, func(i, j int) bool {
		return expectedMetrics[i].ID > expectedMetrics[j].ID
	})

	actualMetrics, err := ms.GetBatch()
	assert.NoError(t, err)

	sort.Slice(actualMetrics, func(i, j int) bool {
		return actualMetrics[i].ID > actualMetrics[j].ID
	})

	assert.Equal(t, expectedMetrics, actualMetrics)
}

func Test_UpdateMetric(t *testing.T) {
	var delta1 int64 = 100
	var delta2 int64 = 200
	var value1 float64 = 10
	var value2 float64 = 20

	ms := New("")

	tests := []struct {
		Input         *metric.Metric
		ExpectedError error
		Name          string
		ExpectedDelta int64
		ExpectedValue float64
	}{
		{
			Name: "first update",
			Input: &metric.Metric{
				ID:    "metric",
				Delta: &delta1,
				Value: &value1,
			},
			ExpectedDelta: delta1,
			ExpectedValue: value1,
			ExpectedError: nil,
		},
		{
			Name: "second update",
			Input: &metric.Metric{
				ID:    "metric",
				Delta: &delta2,
				Value: &value2,
			},
			ExpectedDelta: delta1 + delta2,
			ExpectedValue: value2,
			ExpectedError: nil,
		},
		{
			Name: "empty id",
			Input: &metric.Metric{
				ID: "",
			},
			ExpectedError: metric.ErrCannotUpdateInvalidFormat,
		},
		{
			Name:          "empty metric struct",
			Input:         nil,
			ExpectedError: metric.ErrCannotUpdateInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			assert.ErrorIs(t, ms.UpdateMetric(tt.Input), tt.ExpectedError)

			if tt.Input != nil {
				if tt.Input.ID != "" {
					assert.Equal(t, tt.ExpectedDelta, *ms.metrics[tt.Input.ID].Delta)
					assert.Equal(t, tt.ExpectedValue, *ms.metrics[tt.Input.ID].Value)
				}
			}
		})
	}
}

func Test_UpdateBatch(t *testing.T) {
	ms := New("")
	expectedMetrics := []*metric.Metric{}
	actualMetrics := []*metric.Metric{}

	for i := 0; i < 10; i++ {
		delta := int64(3 * i)
		value := float64(4 * i)
		m := metric.Metric{
			ID:    "metric" + fmt.Sprint(i),
			MType: "type" + fmt.Sprint(2*i),
			Delta: &delta,
			Value: &value,
		}

		expectedMetrics = append(expectedMetrics, &m)
	}

	assert.NoError(t, ms.UpdateBatch(expectedMetrics))

	for i := 0; i < 10; i++ {
		actualMetrics = append(actualMetrics, ms.metrics["metric"+fmt.Sprint(i)])
	}

	assert.Equal(t, expectedMetrics, actualMetrics)
	assert.ErrorIs(t, ms.UpdateBatch([]*metric.Metric{{}}), metric.ErrCannotUpdateInvalidFormat)
}

func Test_AccessCheck(t *testing.T) {
	ms := New("")
	msEmpty := &fileStorage{}

	assert.NoError(t, ms.AccessCheck())
	assert.ErrorIs(t, msEmpty.AccessCheck(), metric.ErrStorageIsNotInitialized)
}

func Test_LoadStorage(t *testing.T) {
	path := "./test.json"
	msUp := New(path)
	msDown := New(path)

	for i := 0; i < 10; i++ {
		delta := int64(3 * i)
		value := float64(4 * i)
		m := metric.Metric{
			ID:    "metric" + fmt.Sprint(i),
			MType: "type" + fmt.Sprint(2*i),
			Delta: &delta,
			Value: &value,
		}

		msUp.metrics[m.ID] = &m
	}

	assert.NoError(t, msUp.UploadStorage())
	assert.NoError(t, msDown.DownloadStorage())

	assert.Equal(t, msUp.metrics, msDown.metrics)

	assert.Error(t, New("").UploadStorage())
	assert.Error(t, New("").DownloadStorage())

	if er := os.Remove(path); er != nil {
		return
	}
}
