package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type gzipLoggerWriter struct {
	http.ResponseWriter
	gz     *gzip.Writer
	status int
	size   int
}

func (w *gzipLoggerWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipLoggerWriter) Write(b []byte) (int, error) {
	if w.gz != nil {
		n, err := w.gz.Write(b)
		w.size += n
		return n, err
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func GzipLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		if r.Header.Get("Content-Encoding") == "gzip" {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "ошибка распаковки gzip", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = gzReader
		}

		rw := &gzipLoggerWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		ae := r.Header.Get("Accept-Encoding")
		ct := w.Header().Get("Content-Type")
		if strings.Contains(ae, "gzip") &&
			(strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/html")) {

			gz := gzip.NewWriter(w)
			defer gz.Close()
			rw.gz = gz
			w.Header().Set("Content-Encoding", "gzip")
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		log.Info().
			Str("method", r.Method).
			Str("uri", r.RequestURI).
			Int("status", rw.status).
			Int("size", rw.size).
			Dur("duration", duration).
			Msg("request handled")
	})
}
