package utils

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

type ResponseWriter struct {
	http.ResponseWriter
	w            http.ResponseWriter
	statusCode   int
	ReqBodyBytes []byte
	ResBodyBytes []byte
}

func NewResponseWriter(w http.ResponseWriter, r *http.Request) *ResponseWriter {
	reqBodyBytes, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(reqBodyBytes))

	return &ResponseWriter{
		w:            w,
		ReqBodyBytes: reqBodyBytes,
		statusCode:   http.StatusOK,
	}
}

func (rw *ResponseWriter) GetStatusCode() int {
	return rw.statusCode
}

func (rw *ResponseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.w.WriteHeader(rw.statusCode)
}

func (rw *ResponseWriter) Write(bytes []byte) (int, error) {
	rw.ResBodyBytes = bytes
	return rw.w.Write(bytes)
}
