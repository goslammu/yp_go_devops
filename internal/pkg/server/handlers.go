package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/go-chi/chi"
)

var (
	errUnsupportedType    = errors.New("unsupported type")
	errInconsistentHashes = errors.New("inconsistent hashes")
	errInvalidFormat      = errors.New("invalid format")
)

const (
	templateHandlerGetAll = "METRICS LIST: <p>{{range .}}{{.ID}}: {{.Value}}{{.Delta}} ({{.MType}})</p>{{end}}"
	storageIsAvailable    = "STORAGE IS AVAILABLE"
)

// List of metric types which could be handled and storaged by server.
var supportedTypes = [...]string{
	Gauge,
	Counter,
}

const (
	Gauge   = "gauge"
	Counter = "counter"
)

const (
	TextPlainCT = "text/plain"
	JSONCT      = "application/json"
	HTTPStr     = "http://"
)

// Checks connection from server to storage.
func (srv *server) handlerCheckConnection(w http.ResponseWriter, r *http.Request) {
	if err := srv.storage.AccessCheck(); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write([]byte(storageIsAvailable)); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Updates batch of metrics kept in request body in json-format.
func (srv *server) handlerUpdateBatch(w http.ResponseWriter, r *http.Request) {
	batch := []*metric.Metric{}

	mj, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(mj, &batch); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for i := range batch {
		if batch[i].Hash == "" {
			log.Println(errInvalidFormat.Error())
			http.Error(w, errInvalidFormat.Error(), http.StatusBadRequest)
			return
		}
		if _, err := srv.checkHash(batch[i]); err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		batch[i].Hash = ""
	}

	if err := srv.storage.UpdateBatch(batch); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.syncUpload != nil {
		srv.fileUpload()
	}
}

// Updates individual metric kept in request body in json-format.
func (srv *server) handlerUpdateJSON(w http.ResponseWriter, r *http.Request) {
	mj, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := metric.Metric{}
	if err := json.Unmarshal(mj, &m); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Header.Get("Hash") != "" && srv.config.HashKey != "" {
		resHash, err := srv.checkHash(&m)
		w.Header().Set("Hash", resHash)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	m.Hash = ""

	if err := checkTypeSupport(m.MType); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	if err := srv.storage.UpdateMetric(&m); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.syncUpload != nil {
		srv.fileUpload()
	}
}

// Updates individual metric kept in URL in format "/type/id/value".
func (srv *server) handlerUpdateDirect(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	if err := checkTypeSupport(mType); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mVal := chi.URLParam(r, "val")
	var mValue float64
	var mDelta int64
	var err error
	switch mType {
	case Gauge:
		mValue, err = strconv.ParseFloat(mVal, 64)
	case Counter:
		mDelta, err = strconv.ParseInt(mVal, 10, 64)
	}
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mName := chi.URLParam(r, "name")

	if err := srv.storage.UpdateMetric(&metric.Metric{
		ID:    mName,
		MType: mType,
		Value: &mValue,
		Delta: &mDelta,
	}); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.syncUpload != nil {
		srv.fileUpload()
	}
}

// Outputs all stored metrics.
func (srv *server) handlerGetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.New("").Parse(templateHandlerGetAll)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	allMetrics, err := srv.storage.GetBatch()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Slice(allMetrics, func(i, j int) bool {
		return allMetrics[i].ID < allMetrics[j].ID
	})

	if er := t.Execute(w, allMetrics); er != nil {
		log.Println(er.Error())
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
}

// Outputs individual metric to response body in json-format.
func (srv *server) handlerGetMetricJSON(w http.ResponseWriter, r *http.Request) {
	mjReq, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mReq := metric.Metric{}
	if er := json.Unmarshal(mjReq, &mReq); er != nil {
		log.Println(er.Error())
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}

	if er := checkTypeSupport(mReq.MType); er != nil {
		log.Println(er.Error())
		http.Error(w, er.Error(), http.StatusNotImplemented)
		return
	}

	mRes, err := srv.storage.GetMetric(mReq.ID)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if mRes.MType != mReq.MType {
		http.Error(w, "cannot get: metric <"+mReq.ID+"> is not <"+mReq.MType+">", http.StatusNotFound)
		return
	}

	if er := mRes.UpdateHash(srv.config.HashKey); er != nil {
		log.Println(er.Error())
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}

	mjRes, err := json.Marshal(mRes)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", JSONCT)

	if _, er := w.Write(mjRes); er != nil {
		log.Println(er.Error())
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
}

// Outputs value of requested metric. Request is parsed from URL in format "/type/name".
func (srv *server) handlerGetMetric(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	if err := checkTypeSupport(mType); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mName := chi.URLParam(r, "name")
	m, err := srv.storage.GetMetric(mName)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if m.MType != mType {
		er := errors.New("cannot get: metric <" + mName + "> is not <" + mType + ">")
		log.Println(er.Error())
		http.Error(w, er.Error(), http.StatusNotFound)
		return
	}

	switch m.MType {
	case Gauge:
		_, err = w.Write([]byte(strconv.FormatFloat(*m.Value, 'f', 3, 64)))
	case Counter:
		_, err = w.Write([]byte(strconv.FormatInt(*m.Delta, 10)))
	}
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Checks if metric type is supported by server or not.
func checkTypeSupport(mType string) error {
	for _, v := range supportedTypes {
		if mType == v {
			return nil
		}
	}

	return errUnsupportedType
}

// Calculates input metric's hash again by server own key and compares it with existing.
// If hashes are inconsistent, returns corresponding error.
func (srv *server) checkHash(m *metric.Metric) (string, error) {
	h := m.Hash

	if er := m.UpdateHash(srv.config.HashKey); er != nil {
		return "", er
	}

	if h != m.Hash {
		return "", errInconsistentHashes
	}

	return m.Hash, nil
}

func (srv *server) fileUpload() {
	srv.syncUpload <- struct{}{}
}
