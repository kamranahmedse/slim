package proxy

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type basicResponseWriter struct {
	header http.Header
	status int
}

func (w *basicResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *basicResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *basicResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

type flushHijackWriter struct {
	*basicResponseWriter
	flushed bool
}

func (w *flushHijackWriter) Flush() {
	w.flushed = true
}

func (w *flushHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	connA, connB := net.Pipe()
	rw := bufio.NewReadWriter(bufio.NewReader(connA), bufio.NewWriter(connA))
	_ = connB.Close()
	return connA, rw, nil
}

func TestStatusRecorderWriteHeaderKeepsFirstStatus(t *testing.T) {
	base := &basicResponseWriter{}
	rec := &statusRecorder{ResponseWriter: base, status: http.StatusOK}

	rec.WriteHeader(http.StatusCreated)
	rec.WriteHeader(http.StatusBadGateway)

	if rec.status != http.StatusCreated {
		t.Fatalf("expected recorder status %d, got %d", http.StatusCreated, rec.status)
	}
}

func TestStatusRecorderFlushDelegatesWhenSupported(t *testing.T) {
	base := &flushHijackWriter{basicResponseWriter: &basicResponseWriter{}}
	rec := &statusRecorder{ResponseWriter: base, status: http.StatusOK}

	rec.Flush()
	if !base.flushed {
		t.Fatal("expected Flush to delegate to underlying writer")
	}
}

func TestStatusRecorderFlushNoopWhenUnsupported(t *testing.T) {
	base := &basicResponseWriter{}
	rec := &statusRecorder{ResponseWriter: base, status: http.StatusOK}

	// Should not panic when flusher is unsupported.
	rec.Flush()
}

func TestStatusRecorderHijackUnsupported(t *testing.T) {
	base := &basicResponseWriter{}
	rec := &statusRecorder{ResponseWriter: base, status: http.StatusOK}

	_, _, err := rec.Hijack()
	if err == nil {
		t.Fatal("expected Hijack to fail when unsupported")
	}
	if !strings.Contains(err.Error(), "does not support hijacking") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusRecorderHijackSupported(t *testing.T) {
	base := &flushHijackWriter{basicResponseWriter: &basicResponseWriter{}}
	rec := &statusRecorder{ResponseWriter: base, status: http.StatusOK}

	conn, _, err := rec.Hijack()
	if err != nil {
		t.Fatalf("expected Hijack success, got: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	_ = conn.Close()
}

func TestStatusRecorderWorksWithHTTPRecorder(t *testing.T) {
	base := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: base, status: http.StatusOK}

	rec.WriteHeader(http.StatusAccepted)
	if rec.status != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.status)
	}
}
