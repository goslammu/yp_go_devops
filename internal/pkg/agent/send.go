package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

const (
	TextPlainCT = "text/plain"
	JSONCT      = "application/json"
	HTTP        = "http://"
	HTTPS       = "https://"
)

// Sends individual metric to the server.
func (agn *agent) sendMetric(name string) error {
	var url, val string
	var body []byte

	m, err := agn.storage.GetMetric(name)
	if err != nil {
		return err
	}
	if errUpdateHash := m.UpdateHash(agn.config.HashKey); errUpdateHash != nil {
		return errUpdateHash
	}

	switch agn.config.ContentType {
	case TextPlainCT:
		switch m.MType {
		case Gauge:
			val = strconv.FormatFloat(*m.Value, 'f', 3, 64)
		case Counter:
			val = strconv.FormatInt(*m.Delta, 10)
		default:
			return fmt.Errorf("cannot send: unsupported metric type <%v>", m.MType)
		}
		url = agn.config.ServerAddress + "/update/" + m.MType + "/" + m.ID + "/" + val
		body = nil
	case JSONCT:
		tmpBody, errMarshal := json.Marshal(m)
		if errMarshal != nil {
			return errMarshal
		}
		url = agn.config.ServerAddress + "/update/"
		body = tmpBody
	default:
		return fmt.Errorf("cannot send: unsupported content type <%v>", agn.config.ContentType)
	}
	res, err := agn.postRequest(url, m.Hash, body)
	if err != nil {
		return err
	}
	defer func() {
		if errBodyClose := res.Body.Close(); errBodyClose != nil {
			log.Println(errBodyClose)
		}
	}()

	if m.MType == Counter {
		if err := agn.resetCounter(name); err != nil {
			return err
		}
	}

	log.Println("SEND METRIC: ", res.Status, res.Request.URL)

	return nil
}

// Sends all storaged metrics collected in batch to the server.
func (agn *agent) sendBatch() error {
	body, err := agn.getStorageBatch()
	if err != nil {
		return err
	}
	res, err := agn.postRequest(agn.config.ServerAddress+"/updates/", "", body)
	if err != nil {
		return err
	}
	defer func() {
		if errCloseBody := res.Body.Close(); errCloseBody != nil {
			log.Println(errCloseBody)
		}
	}()

	if err := agn.resetCounters(); err != nil {
		return err
	}

	log.Println("SEND BATCH: ", res.Status, res.Request.URL)

	return nil
}

// Unified POST-request for all sending methods.
func (agn *agent) postRequest(url, hash string, body []byte) (*http.Response, error) {
	modePrefix := ""

	if agn.config.EnableHTTPS {
		modePrefix = HTTPS
	} else {
		modePrefix = HTTP
	}

	req, err := http.NewRequest("POST", modePrefix+url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", agn.config.ContentType)

	if hash != "" {
		req.Header.Set("Hash", hash)
	}

	return agn.client.Do(req)
}
