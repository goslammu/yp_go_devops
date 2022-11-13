package server

import (
	"flag"
	"fmt"
	"testing"

	"github.com/caarlos0/env"
	"github.com/dcaiman/YP_GO/internal/pkg/filestorage"
	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/dcaiman/YP_GO/internal/pkg/pgxstorage"
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

func Benchmark_filestorageUpdateMetric(b *testing.B) {
	storage := filestorage.New("")

	srv := server{
		storage: storage,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		value := float64(i) * 10
		m := &metric.Metric{
			ID:    "metric" + fmt.Sprint(i),
			MType: Gauge,
			Value: &value,
		}

		b.StartTimer()

		assert.NoError(b, srv.storage.UpdateMetric(m))
	}
}

func Benchmark_filestorageUpdateBatch(b *testing.B) {
	storage := filestorage.New("")

	srv := server{
		storage: storage,
	}

	b.ResetTimer()

	for i := 1; i < b.N; i++ {
		b.StopTimer()

		batch := make([]*metric.Metric, i)

		for j := 0; j < len(batch); j++ {
			value := float64(i) * 10

			batch[j] = &metric.Metric{
				ID:    "metric" + fmt.Sprint(j),
				MType: Gauge,
				Value: &value,
			}
		}

		b.StartTimer()

		assert.NoError(b, srv.storage.UpdateBatch(batch))
	}
}

func Benchmark_pgxstorageUpdateMetric(b *testing.B) {
	storage, err := pgxstorage.New(config.DBAddress, true)
	if err != nil {
		b.Logf("unable to connect to postgre: %v\n", err)
		b.SkipNow()
	}

	srv := server{
		storage: storage,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		value := float64(i) * 10
		m := &metric.Metric{
			ID:    "metric" + fmt.Sprint(i),
			MType: Gauge,
			Value: &value,
		}

		b.StartTimer()

		assert.NoError(b, srv.storage.UpdateMetric(m))
	}
}

func Benchmark_pgxstorageUpdateBatch(b *testing.B) {
	storage, err := pgxstorage.New(config.DBAddress, true)
	if err != nil {
		b.Logf("unable to connect to postgre: %v\n", err)
		b.SkipNow()
	}

	srv := server{
		storage: storage,
	}

	b.ResetTimer()

	for i := 1; i < b.N; i++ {
		b.StopTimer()

		batch := make([]*metric.Metric, i)

		for j := 0; j < len(batch); j++ {
			value := float64(i) * 10

			batch[j] = &metric.Metric{
				ID:    "metric" + fmt.Sprint(j),
				MType: Gauge,
				Value: &value,
			}
		}

		b.StartTimer()

		assert.NoError(b, srv.storage.UpdateBatch(batch))
	}
}
