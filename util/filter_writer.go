package util

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

var terraformHttpConflictBytes = []byte(fmt.Sprintf("%d %s", http.StatusConflict, http.StatusText(http.StatusConflict)))

type FilterWriter struct {
	parent io.Writer
}

func NewFilterWriter(parent io.Writer) *FilterWriter {
	return &FilterWriter{
		parent: parent,
	}
}

func (writer *FilterWriter) Write(p []byte) (n int, err error) {
	if bytes.Contains(p, terraformHttpConflictBytes) {
		return len(p), nil
	}
	return writer.parent.Write(p)
}
