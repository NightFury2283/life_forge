package core_http_response

import (
	"net/http"
)

var (
	statusCodeUninitialized = -1
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     statusCodeUninitialized,
	}
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode != statusCodeUninitialized {
		return
	}
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == statusCodeUninitialized {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *ResponseWriter) GetStatusCodeOrPanic() int {
	if w.statusCode == statusCodeUninitialized {
		return http.StatusOK
	}

	return w.statusCode
}
