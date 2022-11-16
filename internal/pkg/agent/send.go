package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

const (
	TextPlainCT = "text/plain"
	JSONCT      = "application/json"
	HTTPStr     = "http://"
)

// Sends individual metric to the server.
func (agn *agent) sendMetric(name string) error {
	var url, val string
	var body []byte

	m, err := agn.storage.GetMetric(name)
	if err != nil {
		return err
	}
	if erUpdateHash := m.UpdateHash(agn.config.HashKey); erUpdateHash != nil {
		return erUpdateHash
	}

	switch agn.config.ContentType {
	case TextPlainCT:
		switch m.MType {
		case Gauge:
			val = strconv.FormatFloat(*m.Value, 'f', 3, 64)
		case Counter:
			val = strconv.FormatInt(*m.Delta, 10)
		default:
			return errors.New("cannot send: unsupported metric type <" + m.MType + ">")
		}
		url = agn.config.ServerAddr + "/update/" + m.MType + "/" + m.ID + "/" + val
		body = nil
	case JSONCT:
		tmpBody, errMarshal := json.Marshal(m)
		if errMarshal != nil {
			return errMarshal
		}
		url = agn.config.ServerAddr + "/update/"
		body = tmpBody
	default:
		return errors.New("cannot send: unsupported content type <" + agn.config.ContentType + ">")
	}
	res, err := customPostRequest(HTTPStr+url, agn.config.ContentType, m.Hash, bytes.NewBuffer(body))
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
	res, err := customPostRequest(HTTPStr+agn.config.ServerAddr+"/updates/", JSONCT, "", bytes.NewBuffer(body))
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
func customPostRequest(url, contentType, hash string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	if hash != "" {
		req.Header.Set("Hash", hash)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return client, nil
}
