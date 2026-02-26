package daemon

import "encoding/json"

type MessageType string

const (
	MsgShutdown MessageType = "shutdown"
	MsgStatus   MessageType = "status"
	MsgReload   MessageType = "reload"
)

type Request struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type Response struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

type StatusData struct {
	Running bool         `json:"running"`
	PID     int          `json:"pid"`
	Domains []DomainInfo `json:"domains"`
}

type DomainInfo struct {
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Healthy bool   `json:"healthy"`
}

