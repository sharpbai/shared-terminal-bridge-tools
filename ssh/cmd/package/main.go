package main

import (
	"archive/zip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	powerShellVersion = "7.6.6"
	powerShellSHA256  = "02fe458be20493fbdf43f61ea20610b811ee6c738ab1676c61b9cfcd1a33c860"
)

func main() {
	zipPath := flag.String("pwsh-zip", "vendor-pwsh/PowerShell-7.6.6-win-x64.zip", "official Windows x64 PowerShell ZIP")
	flag.Parse()
	if err := packagePair(*zipPath); err != nil {
		fmt.Fprintln(os.Stderr, "package:", err)
		os.Exit(1)
	}
}

func packagePair(zipPath string) error {
	if err := verifyPowerShell(zipPath); err != nil {
		return err
	}
	key, err := loadOrCreateClientKey()
	if err != nil {
		return err
	}
	privatePEM, err := ssh.MarshalPrivateKey(key, "stb-tool-ssh private key")
	if err != nil {
		return err
	}
	public, err := ssh.NewPublicKey(key.Public())
	if err != nil {
		return err
	}
	publicLine := string(ssh.MarshalAuthorizedKey(public))
	generated := "package builddata\n\nfunc init() {\n" +
		"AuthorizedPublicKey = " + strconv.Quote(publicLine) + "\n" +
		"ClientPrivateKey = " + strconv.Quote(string(pem.EncodeToMemory(privatePEM))) + "\n" +
		"PowerShellVersion = " + strconv.Quote(powerShellVersion) + "\n" +
		"PowerShellSHA256 = " + strconv.Quote(powerShellSHA256) + "\n}\n"
	if err := os.WriteFile("internal/builddata/generated.go", []byte(generated), 0600); err != nil {
		return err
	}
	defer os.Remove("internal/builddata/generated.go")
	if err := os.MkdirAll("../dist/ssh", 0700); err != nil {
		return err
	}
	if err := copyFile(zipPath, "cmd/stb-tool-sshd/payload/pwsh.zip"); err != nil {
		return err
	}
	defer os.Remove("cmd/stb-tool-sshd/payload/pwsh.zip")
	if err := build("darwin", "arm64", "../dist/ssh/stb-tool-ssh", "./cmd/stb-tool-ssh"); err != nil {
		return err
	}
	if err := build("windows", "amd64", "../dist/ssh/stb-tool-sshd.exe", "./cmd/stb-tool-sshd"); err != nil {
		return err
	}
	manifest := fmt.Sprintf("PowerShell: %s (Windows x64)\nPowerShell ZIP SHA256: %s\nClient public key fingerprint: %s\n",
		powerShellVersion, powerShellSHA256, ssh.FingerprintSHA256(public))
	if err := os.WriteFile("../dist/ssh/BUILD-INFO.txt", []byte(manifest), 0600); err != nil {
		return err
	}
	fmt.Print(manifest)
	fmt.Println("Built ../dist/ssh/stb-tool-ssh and ../dist/ssh/stb-tool-sshd.exe")
	return nil
}

func verifyPowerShell(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), powerShellSHA256) {
		return errors.New("PowerShell ZIP does not match the official release SHA256")
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return err
	}
	for _, entry := range archive.File {
		if entry.Name == "pwsh.exe" {
			return nil
		}
	}
	return errors.New("PowerShell ZIP has no pwsh.exe")
}

func loadOrCreateClientKey() (ed25519.PrivateKey, error) {
	path := filepath.Join("private", "client_ed25519")
	if data, err := os.ReadFile(path); err == nil {
		parsed, err := ssh.ParseRawPrivateKey(data)
		if err != nil {
			return nil, err
		}
		key, ok := parsed.(*ed25519.PrivateKey)
		if !ok {
			return nil, errors.New("stored client key is not Ed25519")
		}
		return *key, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	block, err := ssh.MarshalPrivateKey(key, "stb-tool-ssh private key")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		return nil, err
	}
	return key, nil
}

func copyFile(from, to string) error {
	input, err := os.Open(from)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func build(goos, goarch, output, pkg string) error {
	command := exec.Command("go", "build", "-trimpath", "-o", output, pkg)
	command.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	return command.Run()
}
