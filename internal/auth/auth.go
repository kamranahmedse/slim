package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/httperr"
)

const apiBase = "https://app.slim.sh"

type Info struct {
	Token string `json:"token"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func Login() (*Info, error) {
	existing, _ := LoadAuth()
	if existing != nil && validateToken(existing.Token) {
		return existing, nil
	}

	if existing != nil {
		_ = Logout()
	}

	return startOAuthLogin()
}

func validateToken(token string) bool {
	req, err := http.NewRequest("GET", apiBase+"/api/me", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func startOAuthLogin() (*Info, error) {
	resp, err := http.Post(apiBase+"/api/auth/cli", "application/json", nil)
	if err != nil {
		return nil, httperr.Wrap("failed to start login", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to start login: %w", httperr.FromResponse(resp))
	}

	var cliResp struct {
		Code string `json:"code"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cliResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Opening browser to log in...")
	if err := openBrowser(cliResp.URL); err != nil {
		fmt.Printf("Could not open browser. Please visit:\n  %s\n", cliResp.URL)
	}

	return pollForCompletion(cliResp.Code)
}

func pollForCompletion(code string) (*Info, error) {
	fmt.Println("Waiting for authentication...")
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	var lastPollErr error
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		pollResp, err := client.Get(fmt.Sprintf("%s/api/auth/cli/poll?code=%s", apiBase, code))
		if err != nil {
			lastPollErr = err
			continue
		}

		if pollResp.StatusCode != http.StatusOK {
			lastPollErr = httperr.FromResponse(pollResp)
			pollResp.Body.Close()
			continue
		}

		var result struct {
			Status string `json:"status"`
			Token  string `json:"token"`
			User   struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"user"`
		}
		json.NewDecoder(pollResp.Body).Decode(&result)
		pollResp.Body.Close()

		if result.Status != "complete" {
			continue
		}

		auth := Info{
			Token: result.Token,
			Name:  result.User.Name,
			Email: result.User.Email,
		}

		if err := saveAuth(auth); err != nil {
			return nil, fmt.Errorf("failed to save credentials: %w", err)
		}

		return &auth, nil
	}

	if lastPollErr != nil {
		return nil, httperr.Wrap("login failed", lastPollErr)
	}

	return nil, fmt.Errorf("login timed out — please try again")
}

func LoadAuth() (*Info, error) {
	data, err := os.ReadFile(config.AuthPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read auth file: %w", err)
	}

	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse auth file: %w", err)
	}

	return &info, nil
}

func Logout() error {
	err := os.Remove(config.AuthPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove auth file: %w", err)
	}
	return nil
}

func LoadOrCreateToken() (string, error) {
	tokenPath := config.TunnelTokenPath()

	data, err := os.ReadFile(tokenPath)
	if err == nil {
		token := string(data)
		if len(token) > 0 {
			return token, nil
		}
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating tunnel token: %w", err)
	}
	token := hex.EncodeToString(b)

	if err := os.MkdirAll(config.Dir(), 0755); err != nil {
		return "", fmt.Errorf("creating config dir: %w", err)
	}
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return "", fmt.Errorf("writing tunnel token: %w", err)
	}

	return token, nil
}

func saveAuth(auth Info) error {
	if err := os.MkdirAll(config.Dir(), 0755); err != nil {
		return err
	}

	data, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	return os.WriteFile(config.AuthPath(), data, 0600)
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
