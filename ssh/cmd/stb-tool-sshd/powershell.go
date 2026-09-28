//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sharpbai/shared-terminal-bridge-tools/ssh/internal/builddata"
)

//go:embed payload/*
var payload embed.FS

var extractionMu sync.Mutex

func ensurePowerShell() (string, error) {
	extractionMu.Lock()
	defer extractionMu.Unlock()
	if builddata.PowerShellVersion == "" || builddata.PowerShellSHA256 == "" {
		return "", errors.New("PowerShell payload metadata is missing")
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(configDir, "stb-tool-sshd")
	target := filepath.Join(root, "pwsh-"+builddata.PowerShellVersion)
	exe := filepath.Join(target, "pwsh.exe")
	if _, err := os.Stat(exe); err == nil {
		return exe, nil
	}
	data, err := payload.ReadFile("payload/pwsh.zip")
	if err != nil {
		return "", fmt.Errorf("PowerShell ZIP not embedded: %w", err)
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), builddata.PowerShellSHA256) {
		return "", errors.New("embedded PowerShell ZIP checksum mismatch")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	temp, err := os.MkdirTemp(root, "pwsh-unpack-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temp)
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	for _, entry := range archive.File {
		name := filepath.Clean(entry.Name)
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || strings.Contains(name, ":") {
			return "", fmt.Errorf("invalid ZIP path: %q", entry.Name)
		}
		path := filepath.Join(temp, name)
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0700); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return "", err
		}
		input, err := entry.Open()
		if err != nil {
			return "", err
		}
		output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			_ = input.Close()
			return "", err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		_ = input.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	if _, err := os.Stat(filepath.Join(temp, "pwsh.exe")); err != nil {
		return "", errors.New("PowerShell ZIP has no pwsh.exe at its root")
	}
	if err := os.Rename(temp, target); err != nil {
		return "", err
	}
	return exe, nil
}
