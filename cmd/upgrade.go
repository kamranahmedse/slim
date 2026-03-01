package cmd

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade slim to the latest version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := "kamranahmedse/slim"

		fmt.Print("Checking for updates... ")
		tag, err := latestTag(repo)
		if err != nil {
			return fmt.Errorf("failed to check latest version: %w", err)
		}
		latest := strings.TrimPrefix(tag, "v")

		if latest == Version {
			fmt.Printf("already up to date (%s)\n", Version)
			return nil
		}
		fmt.Printf("found %s (current: %s)\n", latest, Version)

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to locate current binary: %w", err)
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			return fmt.Errorf("failed to resolve binary path: %w", err)
		}

		tmpDir, err := os.MkdirTemp("", "slim-upgrade-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		filename := fmt.Sprintf("slim_%s_%s_%s.tar.gz", latest, runtime.GOOS, runtime.GOARCH)
		archiveURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, filename)
		checksumURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/checksums.txt", repo, tag)

		fmt.Print("Downloading archive... ")
		archivePath := filepath.Join(tmpDir, filename)
		if err := downloadFile(archiveURL, archivePath); err != nil {
			return fmt.Errorf("failed to download archive: %w", err)
		}
		fmt.Println("done")

		fmt.Print("Verifying checksum... ")
		if err := verifyChecksum(checksumURL, archivePath, filename); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("ok")

		fmt.Print("Extracting... ")
		binaryPath := filepath.Join(tmpDir, "slim")
		if err := extractBinary(archivePath, binaryPath); err != nil {
			return fmt.Errorf("failed to extract archive: %w", err)
		}
		fmt.Println("done")

		fmt.Print("Replacing binary... ")
		if err := replaceBinary(binaryPath, exe); err != nil {
			return err
		}
		fmt.Println("done")

		fmt.Printf("\nUpgraded slim to %s\n", latest)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

func latestTag(repo string) (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head(fmt.Sprintf("https://github.com/%s/releases/latest", repo))
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no redirect from releases/latest")
	}

	parts := strings.Split(loc, "/tag/")
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected redirect URL: %s", loc)
	}

	return parts[1], nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}

func verifyChecksum(checksumURL, filePath, filename string) error {
	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download checksums: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}

	var expectedHash string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), filename) {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				expectedHash = fields[0]
				break
			}
		}
	}
	if expectedHash == "" {
		return fmt.Errorf("checksum not found for %s", filename)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

func extractBinary(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("slim binary not found in archive")
		}
		if err != nil {
			return err
		}

		if filepath.Base(hdr.Name) != "slim" || hdr.Typeflag != tar.TypeReg {
			continue
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}

		return out.Close()
	}
}

func replaceBinary(srcPath, dstPath string) error {
	dir := filepath.Dir(dstPath)

	tmp, err := os.CreateTemp(dir, ".slim-upgrade-*")
	if err != nil {
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to replace binary: %w", err)
		}
		return replaceBinarySudo(srcPath, dstPath)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	in, err := os.Open(srcPath)
	if err != nil {
		tmp.Close()
		return err
	}

	if _, err := io.Copy(tmp, in); err != nil {
		in.Close()
		tmp.Close()
		return err
	}
	in.Close()

	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, dstPath); err != nil {
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to replace binary: %w", err)
		}
		return replaceBinarySudo(srcPath, dstPath)
	}

	return nil
}

func replaceBinarySudo(srcPath, dstPath string) error {
	sudoCmd := exec.Command("sudo", "install", "-m", "0755", srcPath, dstPath)
	sudoCmd.Stdin = os.Stdin
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	if err := sudoCmd.Run(); err != nil {
		return fmt.Errorf("failed to replace binary with sudo: %w", err)
	}
	return nil
}
