package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"text/template"

	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/go-chi/chi/v5"
)

var (
	ErrUnsupportedType    = errors.New("unsupported type")
	ErrInconsistentHashes = errors.New("inconsistent hashes")
	ErrInvalidFormat      = errors.New("invalid format")
)

const (
	templateHandlerGetAll = "METRICS LIST: <p>{{range .}}{{.ID}}: {{.Value}}{{.Delta}} ({{.MType}})</p>{{end}}"
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
func (srv *Server) handlerCheckConnection(w http.ResponseWriter, r *http.Request) {
	if err := srv.Storage.AccessCheck(); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write([]byte("STORAGE IS AVAILABLE")); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Updates batch of metrics kept in request body in json-format.
func (srv *Server) handlerUpdateBatch(w http.ResponseWriter, r *http.Request) {
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
			log.Println(ErrInvalidFormat.Error())
			http.Error(w, ErrInvalidFormat.Error(), http.StatusBadRequest)
			return
		}
		if _, err := srv.checkHash(batch[i]); err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		batch[i].Hash = ""
	}

	if err := srv.Storage.UpdateBatch(batch); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.SyncUpload != nil {
		srv.syncUpload()
	}
}

// Updates individual metric kept in request body in json-format.
func (srv *Server) handlerUpdateJSON(w http.ResponseWriter, r *http.Request) {
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

	if r.Header.Get("Hash") != "" && srv.Cfg.HashKey != "" {
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

	if err := srv.Storage.UpdateMetric(&m); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.SyncUpload != nil {
		srv.syncUpload()
	}
}

// Updates individual metric kept in URL in format "/type/id/value".
func (srv *Server) handlerUpdateDirect(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	fmt.Println(mType)
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

	if err := srv.Storage.UpdateMetric(&metric.Metric{
		ID:    mName,
		MType: mType,
		Value: &mValue,
		Delta: &mDelta,
	}); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.SyncUpload != nil {
		srv.syncUpload()
	}
}

// Outputs all stored metrics.
func (srv *Server) handlerGetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.New("").Parse(templateHandlerGetAll)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	allMetrics, err := srv.Storage.GetBatch()
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
func (srv *Server) handlerGetMetricJSON(w http.ResponseWriter, r *http.Request) {
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

	mRes, err := srv.Storage.GetMetric(mReq.ID)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if mRes.MType != mReq.MType {
		http.Error(w, "cannot get: metric <"+mReq.ID+"> is not <"+mReq.MType+">", http.StatusNotFound)
		return
	}

	if er := mRes.UpdateHash(srv.Cfg.HashKey); er != nil {
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
func (srv *Server) handlerGetMetric(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	if err := checkTypeSupport(mType); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mName := chi.URLParam(r, "name")
	m, err := srv.Storage.GetMetric(mName)
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
	for i := range supportedTypes {
		if mType == supportedTypes[i] {
			return nil
		}
	}
	return ErrUnsupportedType
}

// Calculates input metric's hash again by server own key and compares it with existing.
// If hashes are inconsistent, returns corresponding error.
func (srv *Server) checkHash(m *metric.Metric) (string, error) {
	h := m.Hash

	if er := m.UpdateHash(srv.Cfg.HashKey); er != nil {
		return "", er
	}

	if h != m.Hash {
		return "", ErrInconsistentHashes
	}

	return m.Hash, nil
}

func (srv *Server) syncUpload() {
	srv.SyncUpload <- struct{}{}
}
