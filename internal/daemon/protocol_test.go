package daemon

import (
	"encoding/json"
	"testing"
)

func TestProtocolRoundTripJSON(t *testing.T) {
	req := Request{
		Type: MsgReload,
		Data: json.RawMessage(`{"log_mode":"minimal"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal Request: %v", err)
	}

	var got Request
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal Request: %v", err)
	}

	if got.Type != MsgReload {
		t.Fatalf("unexpected request type: %q", got.Type)
	}
	if string(got.Data) != `{"log_mode":"minimal"}` {
		t.Fatalf("unexpected request data: %s", string(got.Data))
	}
}

func TestStatusDataJSONTags(t *testing.T) {
	status := StatusData{
		Running: true,
		PID:     1234,
		Domains: []DomainInfo{
			{Name: "myapp", Port: 3000, Healthy: true},
		},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Marshal StatusData: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal StatusData: %v", err)
	}

	if _, ok := decoded["running"]; !ok {
		t.Fatal("expected running key")
	}
	if _, ok := decoded["pid"]; !ok {
		t.Fatal("expected pid key")
	}
	if _, ok := decoded["domains"]; !ok {
		t.Fatal("expected domains key")
	}
}
