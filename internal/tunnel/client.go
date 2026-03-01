package tunnel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kamranahmedse/slim/internal/httperr"
	"github.com/kamranahmedse/slim/internal/log"
	proto "github.com/kamranahmedse/slim/protocol"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type RequestEvent struct {
	Method   string
	Path     string
	Status   int
	Duration time.Duration
}

type ClientOptions struct {
	ServerURL string
	Token     string
	Subdomain string
	LocalPort int
	Password  string
	TTL       time.Duration
	OnRequest func(RequestEvent)
}

type Client struct {
	opts ClientOptions
}

func NewClient(opts ClientOptions) *Client {
	return &Client{opts: opts}
}

func (c *Client) Connect(ctx context.Context) (string, error) {
	conn, url, err := c.dial(ctx)
	if err != nil {
		return "", err
	}

	go c.readLoop(ctx, conn)

	return url, nil
}

func (c *Client) dial(ctx context.Context) (*websocket.Conn, string, error) {
	conn, _, err := websocket.Dial(ctx, c.opts.ServerURL, nil)
	if err != nil {
		return nil, "", httperr.Wrap("dialing tunnel server", err)
	}

	conn.SetReadLimit(10 << 20)

	reg := proto.RegistrationRequest{
		Token:     c.opts.Token,
		Subdomain: c.opts.Subdomain,
		Password:  c.opts.Password,
	}
	if c.opts.TTL > 0 {
		reg.TTL = c.opts.TTL.String()
	}

	if err := wsjson.Write(ctx, conn, reg); err != nil {
		conn.Close(websocket.StatusInternalError, "registration write failed")
		return nil, "", fmt.Errorf("sending registration: %w", err)
	}

	var resp proto.RegistrationResponse
	if err := wsjson.Read(ctx, conn, &resp); err != nil {
		conn.Close(websocket.StatusInternalError, "registration read failed")
		return nil, "", fmt.Errorf("reading registration response: %w", err)
	}

	if !resp.OK {
		conn.Close(websocket.StatusNormalClosure, "registration rejected")
		return nil, "", fmt.Errorf("registration failed: %s", resp.Error)
	}

	return conn, resp.URL, nil
}

func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn) {
	backoff := time.Second

	for {
		err := c.readMessages(ctx, conn)
		if err == nil || ctx.Err() != nil {
			return
		}

		log.Error("tunnel connection lost: %v", err)

		for {
			if ctx.Err() != nil {
				return
			}

			log.Info("reconnecting in %s...", backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}

			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			newConn, _, dialErr := c.dial(ctx)
			if dialErr != nil {
				if strings.Contains(dialErr.Error(), "registration failed:") {
					log.Error("%v", dialErr)
					return
				}
				log.Error("reconnect failed: %v", dialErr)
				continue
			}

			log.Info("reconnected to tunnel server")
			conn = newConn
			backoff = time.Second
			break
		}
	}
}

func (c *Client) errorResponse(port int, err error) *http.Response {
	data := serverDownData{Port: port}
	if err != nil {
		data.Error = err.Error()
	}

	var buf bytes.Buffer
	_ = serverDownTmpl.Execute(&buf, data)

	return &http.Response{
		StatusCode: http.StatusBadGateway,
		Status:     "502 Bad Gateway",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Content-Type":   {"text/html; charset=utf-8"},
			"X-Slim-Error":   {"connection-failed"},
			"Content-Length": {fmt.Sprintf("%d", buf.Len())},
		},
		Body:          io.NopCloser(&buf),
		ContentLength: int64(buf.Len()),
	}
}

func (c *Client) readMessages(ctx context.Context, conn *websocket.Conn) error {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	var wsMu sync.Mutex

	for {
		msgType, frame, err := conn.Read(ctx)
		if err != nil {
			return err
		}

		if msgType != websocket.MessageBinary {
			continue
		}

		requestID, data, err := proto.DecodeFrame(frame)
		if err != nil {
			log.Error("decoding frame: %v", err)
			continue
		}

		req, err := proto.DeserializeRequest(data)
		if err != nil {
			log.Error("deserializing request: %v", err)
			continue
		}

		go c.handleRequest(ctx, conn, &wsMu, httpClient, requestID, req)
	}
}

func (c *Client) handleRequest(ctx context.Context, conn *websocket.Conn, wsMu *sync.Mutex, httpClient *http.Client, requestID uint32, req *http.Request) {
	start := time.Now()

	localURL := fmt.Sprintf("http://localhost:%d%s", c.opts.LocalPort, req.URL.RequestURI())
	localReq, err := http.NewRequestWithContext(ctx, req.Method, localURL, req.Body)
	if err != nil {
		log.Error("creating local request: %v", err)
		return
	}
	localReq.Header = req.Header

	resp, err := httpClient.Do(localReq)
	if err != nil {
		log.Error("forwarding to localhost:%d: %v", c.opts.LocalPort, err)
		resp = c.errorResponse(c.opts.LocalPort, err)
	}
	defer resp.Body.Close()

	respBytes, err := proto.SerializeResponse(resp)
	if err != nil {
		log.Error("serializing response: %v", err)
		return
	}

	frameOut := proto.EncodeFrame(requestID, respBytes)

	wsMu.Lock()
	writeErr := conn.Write(ctx, websocket.MessageBinary, frameOut)
	wsMu.Unlock()

	if writeErr != nil {
		log.Error("writing response frame: %v", writeErr)
		return
	}

	if c.opts.OnRequest != nil {
		c.opts.OnRequest(RequestEvent{
			Method:   req.Method,
			Path:     req.URL.Path,
			Status:   resp.StatusCode,
			Duration: time.Since(start),
		})
	}
}
