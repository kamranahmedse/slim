package protocol

import (
	"net/http"
	"strings"
	"testing"
)

func TestFrameEncodeDecode(t *testing.T) {
	data := []byte("hello world")
	var requestID uint32 = 42

	frame := EncodeFrame(requestID, data)
	gotID, gotData, err := DecodeFrame(frame)
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	if gotID != requestID {
		t.Errorf("requestID = %d, want %d", gotID, requestID)
	}
	if string(gotData) != string(data) {
		t.Errorf("data = %q, want %q", gotData, data)
	}
}

func TestFrameDecodeError(t *testing.T) {
	_, _, err := DecodeFrame([]byte{0, 1})
	if err == nil {
		t.Fatal("expected error for short frame")
	}
}

func TestFrameDecodeEmptyPayload(t *testing.T) {
	frame := EncodeFrame(1, nil)
	id, data, err := DecodeFrame(frame)
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	if id != 1 {
		t.Errorf("requestID = %d, want 1", id)
	}
	if len(data) != 0 {
		t.Errorf("expected empty payload, got %d bytes", len(data))
	}
}

func TestSerializeDeserializeRequest(t *testing.T) {
	original, err := http.NewRequest("POST", "http://localhost/api/test?q=1", strings.NewReader("body"))
	if err != nil {
		t.Fatal(err)
	}
	original.Header.Set("Content-Type", "application/json")
	original.Header.Set("X-Custom", "value")

	data, err := SerializeRequest(original)
	if err != nil {
		t.Fatalf("SerializeRequest: %v", err)
	}

	restored, err := DeserializeRequest(data)
	if err != nil {
		t.Fatalf("DeserializeRequest: %v", err)
	}

	if restored.Method != "POST" {
		t.Errorf("method = %q, want POST", restored.Method)
	}
	if restored.URL.Path != "/api/test" {
		t.Errorf("path = %q, want /api/test", restored.URL.Path)
	}
	if restored.URL.RawQuery != "q=1" {
		t.Errorf("query = %q, want q=1", restored.URL.RawQuery)
	}
	if restored.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", restored.Header.Get("Content-Type"))
	}
	if restored.Header.Get("X-Custom") != "value" {
		t.Errorf("X-Custom = %q, want value", restored.Header.Get("X-Custom"))
	}
}

func TestSerializeDeserializeResponse(t *testing.T) {
	original := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       http.NoBody,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	data, err := SerializeResponse(original)
	if err != nil {
		t.Fatalf("SerializeResponse: %v", err)
	}

	restored, err := DeserializeResponse(data)
	if err != nil {
		t.Fatalf("DeserializeResponse: %v", err)
	}
	defer restored.Body.Close()

	if restored.StatusCode != 200 {
		t.Errorf("status = %d, want 200", restored.StatusCode)
	}
	if restored.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("Content-Type = %q, want text/plain", restored.Header.Get("Content-Type"))
	}
}
