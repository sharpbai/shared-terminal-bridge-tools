//go:build windows

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sharpbai/shared-terminal-bridge-tools/ssh/internal/builddata"
	"golang.org/x/crypto/ssh"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	port := flag.Int("port", 2222, "TCP listening port")
	listenIP := flag.String("listen", "0.0.0.0", "local IP to bind")
	user := flag.String("user", "stb", "logical SSH username")
	flag.Parse()
	if *port < 1 || *port > 65535 || *user == "" {
		return errors.New("invalid port or user")
	}
	if builddata.AuthorizedPublicKey == "" {
		return errors.New("authorized key is absent; build a private pair with cmd/package")
	}
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(builddata.AuthorizedPublicKey))
	if err != nil {
		return fmt.Errorf("embedded public key: %w", err)
	}
	if _, err := ensurePowerShell(); err != nil {
		return fmt.Errorf("embedded PowerShell: %w", err)
	}
	hostSigner, err := loadOrCreateHostKey()
	if err != nil {
		return err
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(meta ssh.ConnMetadata, offered ssh.PublicKey) (*ssh.Permissions, error) {
			if meta.User() != *user || subtle.ConstantTimeCompare(offered.Marshal(), publicKey.Marshal()) != 1 {
				return nil, errors.New("unauthorized")
			}
			return nil, nil
		},
		MaxAuthTries: 3,
	}
	config.AddHostKey(hostSigner)
	address := net.JoinHostPort(*listenIP, strconv.Itoa(*port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Printf("stb-tool-sshd ready on %s\n", address)
	fmt.Printf("SSH user: %s | Windows process user: current user\n", *user)
	fmt.Printf("Host fingerprint: %s\n", ssh.FingerprintSHA256(hostSigner.PublicKey()))
	fmt.Println("Press Ctrl+C to stop. This program does not install a Windows service.")
	for {
		connection, err := listener.Accept()
		if err != nil {
			return err
		}
		go handleConnection(connection, config)
	}
}

func handleConnection(connection net.Conn, config *ssh.ServerConfig) {
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(15 * time.Second))
	serverConn, channels, requests, err := ssh.NewServerConn(connection, config)
	if err != nil {
		log.Printf("SSH handshake from %s: %v", connection.RemoteAddr(), err)
		return
	}
	_ = connection.SetDeadline(time.Time{})
	defer serverConn.Close()
	log.Printf("authenticated %s from %s", serverConn.User(), connection.RemoteAddr())
	go ssh.DiscardRequests(requests)
	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(ssh.UnknownChannelType, "only interactive sessions are supported")
			continue
		}
		stream, streamRequests, err := channel.Accept()
		if err != nil {
			continue
		}
		go handleSession(stream, streamRequests)
	}
}

func loadOrCreateHostKey() (ssh.Signer, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(configDir, "stb-tool-sshd")
	path := filepath.Join(dir, "host_ed25519")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		block, err := ssh.MarshalPrivateKey(privateKey, "stb-tool-sshd host key")
		if err != nil {
			return nil, err
		}
		data = pem.EncodeToMemory(block)
		if err := os.WriteFile(path, data, 0600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(data)
}
