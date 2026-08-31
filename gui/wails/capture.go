package main

import (
	"bytes"
	"net/http"
)

// capture bufferise une réponse HTTP pour permettre la réécriture d'index.html.
type capture struct {
	header http.Header
	buf    bytes.Buffer
	code   int
}

func (c *capture) Header() http.Header         { return c.header }
func (c *capture) WriteHeader(code int)        { c.code = code }
func (c *capture) Write(b []byte) (int, error) { return c.buf.Write(b) }
