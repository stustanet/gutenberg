package main

import (
	"bytes"
	"net/http"
)

var _ http.ResponseWriter = new(ResponseBuffer)

type ResponseBuffer struct {
	buf  *bytes.Buffer
	rw   http.ResponseWriter
	code int
}

func NewResponseBuffer(rw http.ResponseWriter) *ResponseBuffer {
	return &ResponseBuffer{
		buf:  new(bytes.Buffer),
		rw:   rw,
		code: http.StatusOK,
	}
}

func (rb *ResponseBuffer) Header() http.Header {
	return rb.rw.Header()
}

func (rb *ResponseBuffer) Write(p []byte) (int, error) {
	return rb.buf.Write(p)
}

func (rb *ResponseBuffer) WriteHeader(code int) {
	rb.code = code
}

func (rb *ResponseBuffer) Bytes() []byte {
	return rb.buf.Bytes()
}

func (rb *ResponseBuffer) Code() int {
	return rb.code
}
