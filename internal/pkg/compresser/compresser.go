package compresser

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type customWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w customWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Middleware component for handling gzip-encoded requests.
func Compresser(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			defer func() {
				if errReaderClose := gzipReader.Close(); errReaderClose != nil {
					log.Println(errReaderClose)
				}
			}()

			r.Body = gzipReader
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			handler.ServeHTTP(w, r)
			return
		}

		gzipWriter, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer func() {
			if errWriterClose := gzipWriter.Close(); errWriterClose != nil {
				log.Println(errWriterClose)
			}
		}()

		w.Header().Set("Content-Encoding", "gzip")
		handler.ServeHTTP(customWriter{ResponseWriter: w, Writer: gzipWriter}, r)
	})
}
