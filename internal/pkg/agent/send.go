package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/goslammu/yp_go_devops/internal/pkg/metric"
	log "github.com/sirupsen/logrus"
)

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	errUnsupportedMetricType  = errors.New("unsupported metric type")
)

const (
	ContentTypeTextPlain = "text/plain"
	ContentTypeJSON      = "application/json"
	HTTP                 = "http://"
	HTTPS                = "https://"
)

// Sends individual metric to the server.
func (agn *agent) sendMetric(name string) error {
	m, err := agn.storage.GetMetric(name)
	if err != nil {
		return err
	}

	if errUpdateHash := m.UpdateHash(agn.config.HashKey); errUpdateHash != nil {
		return errUpdateHash
	}

	switch agn.config.ContentType {
	case ContentTypeTextPlain:
		if err := agn.sendMetricAsTextPlain(m); err != nil {
			return err
		}
	case ContentTypeJSON:
		if err := agn.sendMetricAsJSON(m); err != nil {
			return err
		}
	default:
		return errUnsupportedContentType
	}

	if m.MType == Counter {
		if err := agn.resetCounter(name); err != nil {
			return err
		}
	}

	return nil
}

func (agn *agent) sendMetricAsTextPlain(m *metric.Metric) error {
	var val string

	switch m.MType {
	case Gauge:
		val = strconv.FormatFloat(*m.Value, 'f', 3, 64)
	case Counter:
		val = strconv.FormatInt(*m.Delta, 10)
	default:
		return errUnsupportedMetricType
	}

	if err := agn.postRequest(
		agn.config.ServerAddress+"/update/"+m.MType+"/"+m.ID+"/"+val,
		m.Hash,
		ContentTypeTextPlain,
		nil); err != nil {

		return err
	}

	return nil
}

func (agn *agent) sendMetricAsJSON(m *metric.Metric) error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	if err := agn.postRequest(
		agn.config.ServerAddress+"/update/",
		m.Hash,
		ContentTypeJSON,
		body); err != nil {
		return err
	}

	return nil
}

// Sends all storaged metrics collected in batch to the server.
func (agn *agent) sendBatchAsJSON() error {
	body, err := agn.getStorageBatch()
	if err != nil {
		return err
	}
	if err := agn.postRequest(
		agn.config.ServerAddress+"/updates/",
		"",
		ContentTypeJSON,
		body); err != nil {
		return err
	}

	if err := agn.resetCounters(); err != nil {
		return err
	}

	return nil
}

// Unified POST-request for all sending methods.
func (agn *agent) postRequest(url, hash, contentType string, body []byte) error {
	modePrefix := ""

	if agn.config.EnableHTTPS {
		modePrefix = HTTPS
	} else {
		modePrefix = HTTP
	}

	req, err := http.NewRequest(
		"POST",
		modePrefix+url,
		bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", contentType)

	if hash != "" {
		req.Header.Set("Hash", hash)
	}

	res, err := agn.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if errBodyClose := res.Body.Close(); errBodyClose != nil {
			log.Println(errBodyClose)
		}
	}()

	log.Println("REQ SENT:", res.Status, res.Request.URL)

	return nil
}
