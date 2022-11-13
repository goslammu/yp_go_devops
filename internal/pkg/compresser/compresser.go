package compresser

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
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
				log.Println(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			defer func() {
				if er := gzipReader.Close(); er != nil {
					log.Println(er.Error())
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
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer func() {
			if er := gzipWriter.Close(); er != nil {
				log.Println(er.Error())
			}
		}()

		w.Header().Set("Content-Encoding", "gzip")
		handler.ServeHTTP(customWriter{ResponseWriter: w, Writer: gzipWriter}, r)
	})
}
