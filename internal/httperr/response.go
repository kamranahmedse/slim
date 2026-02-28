package httperr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func FromResponse(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	var apiErr struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &apiErr) == nil {
		if msg := apiErr.Error; msg != "" {
			return fmt.Errorf("server error: %s (HTTP %d)", msg, resp.StatusCode)
		}
		if msg := apiErr.Message; msg != "" {
			return fmt.Errorf("server error: %s (HTTP %d)", msg, resp.StatusCode)
		}
	}

	if hint := StatusHint(resp.StatusCode); hint != "" {
		return fmt.Errorf("server returned HTTP %d — %s", resp.StatusCode, hint)
	}

	return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
}

func StatusHint(code int) string {
	switch code {
	case 401:
		return "unauthorized, please try logging in again"
	case 403:
		return "access denied"
	case 404:
		return "endpoint not found, you may need to update slim"
	case 429:
		return "too many requests, please wait a moment and try again"
	case 500:
		return "internal server error, please try again later"
	case 502, 503, 521, 522, 523:
		return "the server is temporarily unavailable, please try again later"
	case 504, 524:
		return "the server took too long to respond, please try again later"
	default:
		if code >= 500 {
			return "server error, please try again later"
		}
		return ""
	}
}
