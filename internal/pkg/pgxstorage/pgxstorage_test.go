package pgxstorage

import (
	"flag"
	"fmt"
	"sort"
	"testing"

	"github.com/caarlos0/env"
	"github.com/goslammu/yp_go_devops/internal/pkg/metric"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
)

var config struct {
	DBAddress string `env:"DATABASE_DSN"`
}

func init() {
	config.DBAddress = "postgresql://postgres:1@127.0.0.1:5432"
	flag.StringVar(&config.DBAddress, "d", config.DBAddress, "database address")
	if err := env.Parse(&config); err != nil {
		return
	}
}

func Test_New(t *testing.T) {
	var ms *pgxStorage
	var err error

	id := "id"

	t.Run("incorrect db address", func(t *testing.T) {
		ms, err = New("", true)

		assert.Error(t, err)
		assert.Nil(t, ms)
	})

	t.Run("correct db address", func(t *testing.T) {
		ms, err = New(config.DBAddress, false)
		if err != nil {
			t.Logf("unable to connect to postgre: %v\n", err)
			t.SkipNow()
		}
		assert.NotNil(t, ms)

		_, err = ms.DB.Exec(stUpdateMetric, id, "", 0, 0)
		assert.NoError(t, err)

		m := metric.Metric{}

		row := ms.DB.QueryRow(stGetMetric, id)
		err = row.Scan(&m.ID, &m.MType, &m.Value, &m.Delta)
		assert.NoError(t, err)
		assert.Equal(t, id, m.ID)

		assert.NoError(t, ms.DB.Close())
	})

	t.Run("drop db", func(t *testing.T) {
		ms, err = New(config.DBAddress, true)
		if err != nil {
			t.Logf("unable to connect to postgre: %v\n", err)
			t.SkipNow()
		}
		assert.NotNil(t, ms)

		m := metric.Metric{}

		row := ms.DB.QueryRow(stGetMetric, id)
		err = row.Scan(&m.ID, &m.MType, &m.Value, &m.Delta)
		assert.Error(t, err)

		assert.NoError(t, ms.DB.Close())
	})

}

func Test_Close(t *testing.T) {
	ms, err := New(config.DBAddress, false)
	if err != nil {
		t.Logf("unable to connect to postgre: %v\n", err)
		t.SkipNow()
	}
	assert.NotNil(t, ms)

	assert.NoError(t, ms.DB.Ping())

	assert.NoError(t, ms.Close())

	assert.Error(t, ms.DB.Ping())
}

func Test_AccessCheck(t *testing.T) {
	ms, err := New(config.DBAddress, false)
	if err != nil {
		t.Logf("unable to connect to postgre: %v\n", err)
		t.SkipNow()
	}
	assert.NotNil(t, ms)

	assert.NoError(t, ms.AccessCheck())

	assert.NoError(t, ms.DB.Close())

	assert.Error(t, ms.AccessCheck())

	assert.NoError(t, ms.DB.Close())
}

func Test_GetMetric(t *testing.T) {
	var delta int64 = 100
	var value float64 = 200

	ms, err := New(config.DBAddress, true)
	if err != nil {
		t.Logf("unable to connect to postgre: %v\n", err)
		t.SkipNow()
	}
	assert.NotNil(t, ms)

	m := metric.Metric{
		ID:    "id",
		MType: "type",
		Delta: &delta,
		Value: &value,
	}

	_, err = ms.DB.Exec(stUpdateMetric, &m.ID, &m.MType, &m.Value, &m.Delta)
	assert.NoError(t, err)

	tests := []struct {
		ExpectedOutput *metric.Metric
		ExpectedError  error
		Name           string
		Input          string
	}{
		{
			Name:           "existent metric",
			Input:          m.ID,
			ExpectedOutput: &m,
			ExpectedError:  nil,
		},
		{
			Name:           "non-existent metric",
			Input:          "any",
			ExpectedOutput: nil,
			ExpectedError:  metric.ErrMetricDoesntExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			m, err := ms.GetMetric(tt.Input)

			assert.ErrorIs(t, err, tt.ExpectedError)
			assert.Equal(t, tt.ExpectedOutput, m)
		})
	}

	assert.NoError(t, ms.DB.Close())
}

func Test_GetBatch(t *testing.T) {
	ms, err := New(config.DBAddress, true)
	if err != nil {
		t.Logf("unable to connect to postgre: %v\n", err)
		t.SkipNow()
	}
	assert.NotNil(t, ms)

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
		_, err = ms.DB.Exec(stUpdateMetric, &m.ID, &m.MType, &m.Value, &m.Delta)
		assert.NoError(t, err)
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

	assert.NoError(t, ms.DB.Close())
}

func Test_UpdateMetric(t *testing.T) {
	ms, err := New(config.DBAddress, true)
	if err != nil {
		t.Logf("unable to connect to postgre: %v\n", err)
		t.SkipNow()
	}
	assert.NotNil(t, ms)

	var delta1 int64 = 100
	var delta2 int64 = 200
	var value1 float64 = 10
	var value2 float64 = 20

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
			Name:          "nil input",
			Input:         nil,
			ExpectedError: metric.ErrCannotUpdateInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			assert.ErrorIs(t, ms.UpdateMetric(tt.Input), tt.ExpectedError)

			if tt.Name == "empty id" || tt.Name == "nil input" {
				t.Skip()
			}
			m := metric.Metric{}

			row := ms.DB.QueryRow(stGetMetric, tt.Input.ID)
			err = row.Scan(&m.ID, &m.MType, &m.Value, &m.Delta)
			assert.NoError(t, err)
			assert.Equal(t, tt.Input.ID, m.ID)

			assert.Equal(t, tt.ExpectedDelta, *m.Delta)
			assert.Equal(t, tt.ExpectedValue, *m.Value)
		})
	}

	assert.NoError(t, ms.DB.Close())
}

func Test_UpdateBatch(t *testing.T) {
	ms, err := New(config.DBAddress, true)
	if err != nil {
		t.Logf("unable to connect to postgre: %v\n", err)
		t.SkipNow()
	}
	assert.NotNil(t, ms)

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
		m := metric.Metric{}

		row := ms.DB.QueryRow(stGetMetric, "metric"+fmt.Sprint(i))
		err = row.Scan(&m.ID, &m.MType, &m.Value, &m.Delta)
		assert.NoError(t, err)

		actualMetrics = append(actualMetrics, &m)
	}

	assert.Equal(t, expectedMetrics, actualMetrics)
	assert.ErrorIs(t, ms.UpdateBatch([]*metric.Metric{{}}), metric.ErrCannotUpdateInvalidFormat)
	assert.ErrorIs(t, ms.UpdateBatch([]*metric.Metric{nil, nil}), metric.ErrCannotUpdateInvalidFormat)

	assert.NoError(t, ms.DB.Close())
}
