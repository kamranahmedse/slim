package tunnel

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httputil"
)

type RegistrationRequest struct {
	Token     string `json:"token"`
	Subdomain string `json:"subdomain"`
	Password  string `json:"password,omitempty"`
	TTL       string `json:"ttl,omitempty"`
}

type RegistrationResponse struct {
	OK        bool   `json:"ok"`
	URL       string `json:"url"`
	Subdomain string `json:"subdomain"`
	Error     string `json:"error,omitempty"`
}

func EncodeFrame(requestID uint32, data []byte) []byte {
	frame := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(frame[:4], requestID)
	copy(frame[4:], data)
	return frame
}

func DecodeFrame(frame []byte) (uint32, []byte, error) {
	if len(frame) < 4 {
		return 0, nil, fmt.Errorf("frame too short: %d bytes", len(frame))
	}
	requestID := binary.BigEndian.Uint32(frame[:4])
	return requestID, frame[4:], nil
}

func SerializeRequest(r *http.Request) ([]byte, error) {
	return httputil.DumpRequest(r, true)
}

func DeserializeRequest(data []byte) (*http.Request, error) {
	return http.ReadRequest(bufio.NewReader(bytes.NewReader(data)))
}

func SerializeResponse(resp *http.Response) ([]byte, error) {
	return httputil.DumpResponse(resp, true)
}

func DeserializeResponse(data []byte) (*http.Response, error) {
	return http.ReadResponse(bufio.NewReader(bytes.NewReader(data)), nil)
}
