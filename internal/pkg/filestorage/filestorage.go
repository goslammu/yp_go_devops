package filestorage

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/dcaiman/YP_GO/internal/pkg/metric"
)

// Realization of metrics storage based on map. Is concurrent-safe due to Mutex.
type fileStorage struct {
	metrics  map[string]*metric.Metric
	FilePath string
	sync.RWMutex
}

// Constructor.
func New(filePath string) *fileStorage {
	return &fileStorage{
		metrics:  map[string]*metric.Metric{},
		FilePath: filePath,
	}
}

func (st *fileStorage) Close() error {
	return nil
}

// Returns existing metric by it's name.
func (st *fileStorage) GetMetric(name string) (*metric.Metric, error) {
	st.Lock()
	defer st.Unlock()

	if m, ok := st.metrics[name]; ok {
		return m, nil
	}
	return nil, metric.ErrMetricDoesntExist
}

// Returns all storaged metrics in slice.
func (st *fileStorage) GetBatch() ([]*metric.Metric, error) {
	st.Lock()
	defer st.Unlock()

	allMetrics := make([]*metric.Metric, len(st.metrics))
	i := 0
	for _, k := range st.metrics {
		allMetrics[i] = k
		i++
	}
	return allMetrics, nil
}

// Updates metric valuable fields: overrides Value and increments Delta.
func (st *fileStorage) UpdateMetric(m *metric.Metric) error {
	st.Lock()
	defer st.Unlock()

	return st.updateMetric(m)
}

func (st *fileStorage) updateMetric(m *metric.Metric) error {
	if m == nil {
		return metric.ErrCannotUpdateInvalidFormat
	}

	if m.ID == "" {
		return metric.ErrCannotUpdateInvalidFormat
	}

	if m.Delta != nil {
		if mEx, ok := st.metrics[m.ID]; ok && mEx.Delta != nil {
			del := *mEx.Delta + *m.Delta
			m.Delta = &del
		}
	}

	st.metrics[m.ID] = m

	return nil
}

// Updates metrics collected in input batch by valuable fields: overrides Values and increments Deltas.
func (st *fileStorage) UpdateBatch(batch []*metric.Metric) error {
	st.Lock()
	defer st.Unlock()

	for i := range batch {
		if err := st.updateMetric(batch[i]); err != nil {
			return err
		}
	}
	return nil
}

// Checks if storage is initialized.
func (st *fileStorage) AccessCheck() error {
	st.Lock()
	defer st.Unlock()

	if st.metrics == nil {
		return metric.ErrStorageIsNotInitialized
	}
	return nil
}

// Uploads storage to json-file on path defined in constructor.
func (st *fileStorage) UploadStorage() error {
	var err error

	st.Lock()
	defer st.Unlock()

	file, err := os.OpenFile(st.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer func() {
		if errFileClose := file.Close(); errFileClose != nil {
			log.Println(errFileClose)
		}
	}()

	for name := range st.metrics {
		mj, err := json.Marshal(st.metrics[name])
		if err != nil {
			return err
		}
		mj = append(mj, '\n')
		_, err = file.Write(mj)
		if err != nil {
			return err
		}
	}

	log.Println("UPLOADED TO: " + st.FilePath)

	return nil
}

// Downlod storage from json-file on path defined in constructor.
func (st *fileStorage) DownloadStorage() error {
	st.Lock()
	defer st.Unlock()

	file, err := os.Open(st.FilePath)
	if err != nil {
		return err
	}
	defer func() {
		if errFileClose := file.Close(); errFileClose != nil {
			log.Println(errFileClose)
		}
	}()

	b := bufio.NewScanner(file)

	for b.Scan() {
		m := metric.Metric{}

		if err := json.Unmarshal(b.Bytes(), &m); err != nil {
			return err
		}

		if err := st.updateMetric(&m); err != nil {
			return err
		}
	}

	log.Println("DOWNLOADED FROM: " + st.FilePath)

	return nil
}
