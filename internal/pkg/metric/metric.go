package metric

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

var (
	ErrStorageIsNotInitialized   = errors.New("storage is not initialized")
	ErrMetricDoesntExist         = errors.New("metric doesn't exist")
	ErrCannotUpdateInvalidFormat = errors.New("cannot update metric: invalid format")
)

// General interface of metric storages used by Agent and Server.
type MetricStorage interface {
	// Returns existing metric by it's name.
	GetMetric(id string) (*Metric, error)

	// Returns all storaged metrics in slice.
	GetBatch() ([]*Metric, error)

	// Updates metric valuable fields: overrides Value and increments Delta.
	UpdateMetric(m *Metric) error

	// Updates metrics collected in input batch by valuable fields: overrides Values and increments Deltas.
	UpdateBatch(batch []*Metric) error

	// Checks if storage is initialized.
	AccessCheck() error
}

type Metric struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

// Refreshes metric's hash by given key.
func (m *Metric) UpdateHash(key string) error {
	if key == "" {
		m.Hash = ""
		return nil
	}

	var deltaPart, valuePart string

	if m.Delta != nil {
		deltaPart = fmt.Sprintf("%s:%s:%d", m.ID, m.MType, *m.Delta)
	}
	if m.Value != nil {
		valuePart = fmt.Sprintf("%s:%s:%f", m.ID, m.MType, *m.Value)
	}

	h := hmac.New(sha256.New, []byte(key))

	if _, err := h.Write([]byte(deltaPart + valuePart)); err != nil {
		return err
	}

	m.Hash = hex.EncodeToString(h.Sum(nil))

	return nil
}
