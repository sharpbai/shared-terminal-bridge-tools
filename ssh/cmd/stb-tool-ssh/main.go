//go:build darwin

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sharpbai/shared-terminal-bridge-tools/ssh/internal/builddata"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "stb-tool-ssh:", err)
		os.Exit(1)
	}
}

func run() error {
	port := flag.Int("port", 2222, "SSH port on the Windows host")
	user := flag.String("user", "stb", "logical SSH username")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: stb-tool-ssh IP [--port 2222] [--user stb]")
		flag.PrintDefaults()
	}
	if err := flag.CommandLine.Parse(orderedArgs(os.Args[1:])); err != nil {
		return err
	}
	if flag.NArg() != 1 || *port < 1 || *port > 65535 || *user == "" {
		flag.Usage()
		return errors.New("invalid address, port, or user")
	}
	if builddata.ClientPrivateKey == "" {
		return errors.New("client key is absent; build a private pair with cmd/package")
	}
	signer, err := ssh.ParsePrivateKey([]byte(builddata.ClientPrivateKey))
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(flag.Arg(0), strconv.Itoa(*port))
	callback, err := hostKeyCallback()
	if err != nil {
		return err
	}
	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            *user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: callback,
		Timeout:         15 * time.Second,
	})
	if err != nil {
		return err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("interactive terminal required")
	}
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width, height = 80, 25
	}
	termName := os.Getenv("TERM")
	if termName == "" {
		termName = "xterm-256color"
	}
	if err := session.RequestPty(termName, height, width, ssh.TerminalModes{}); err != nil {
		return err
	}
	session.Stdout, session.Stderr = os.Stdout, os.Stderr
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	if err := session.Shell(); err != nil {
		return err
	}
	resize := make(chan os.Signal, 1)
	signal.Notify(resize, syscall.SIGWINCH)
	defer signal.Stop(resize)
	go func() {
		for range resize {
			w, h, sizeErr := term.GetSize(int(os.Stdin.Fd()))
			if sizeErr == nil {
				_ = session.WindowChange(h, w)
			}
		}
	}()
	go func() {
		_, _ = io.Copy(stdin, os.Stdin)
		_ = stdin.Close()
	}()
	return session.Wait()
}

// flag's standard parser stops at the first IP. Keep the one-command form
// convenient by accepting options on either side of it.
func orderedArgs(args []string) []string {
	options := make([]string, 0, len(args))
	positionals := make([]string, 0, 1)
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--port" || arg == "-port" || arg == "--user" || arg == "-user" {
			options = append(options, arg)
			if index+1 < len(args) {
				index++
				options = append(options, args[index])
			}
		} else if strings.HasPrefix(arg, "-") {
			options = append(options, arg)
		} else {
			positionals = append(positionals, arg)
		}
	}
	return append(options, positionals...)
}

// The client pins each server host key after explicit confirmation.
func hostKeyCallback() (ssh.HostKeyCallback, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(configDir, "stb-tool-ssh", "known_hosts")
	known := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				known[parts[0]] = parts[1]
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		_ = remote
		fingerprint := ssh.FingerprintSHA256(key)
		if pinned, ok := known[hostname]; ok {
			if pinned != fingerprint {
				return fmt.Errorf("host key changed for %s: got %s, expected %s", hostname, fingerprint, pinned)
			}
			return nil
		}
		fmt.Fprintf(os.Stderr, "New SSH host %s fingerprint: %s\nTrust this host? Type yes: ", hostname, fingerprint)
		answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil || strings.TrimSpace(answer) != "yes" {
			return errors.New("host key not trusted")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = fmt.Fprintf(file, "%s %s\n", hostname, fingerprint)
		return err
	}, nil
}
