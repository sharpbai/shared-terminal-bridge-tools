//go:build windows

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"time"

	pty "github.com/aymanbagabas/go-pty"
	"golang.org/x/crypto/ssh"
)

type windowSize struct{ width, height int }

func handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()
	size := windowSize{80, 25}
	var terminal pty.Pty
	var cancel context.CancelFunc
	var started bool
	var closeOnce sync.Once
	stop := func() {
		closeOnce.Do(func() {
			if cancel != nil {
				cancel()
			}
			if terminal != nil {
				_ = terminal.Close()
			}
		})
	}
	defer stop()
	for request := range requests {
		switch request.Type {
		case "pty-req":
			if started {
				_ = request.Reply(false, nil)
				continue
			}
			if parsed, ok := parsePtyRequest(request.Payload); ok {
				size = parsed
				_ = request.Reply(true, nil)
			} else {
				_ = request.Reply(false, nil)
			}
		case "window-change":
			if parsed, ok := parseWindowChange(request.Payload); ok {
				size = parsed
				if terminal != nil {
					_ = terminal.Resize(size.width, size.height)
				}
			}
		case "shell":
			if started || len(request.Payload) != 0 {
				_ = request.Reply(false, nil)
				continue
			}
			pwshPath, err := ensurePowerShell()
			if err != nil {
				_ = request.Reply(false, nil)
				log.Printf("PowerShell unavailable: %v", err)
				continue
			}
			terminal, err = pty.New()
			if err != nil {
				_ = request.Reply(false, nil)
				log.Printf("ConPTY unavailable: %v", err)
				continue
			}
			_ = terminal.Resize(size.width, size.height)
			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
			command := terminal.CommandContext(ctx, pwshPath, "-NoLogo")
			command.Dir, _ = os.UserHomeDir()
			if err := command.Start(); err != nil {
				_ = request.Reply(false, nil)
				cancel()
				stop()
				log.Printf("PowerShell start failed: %v", err)
				return
			}
			started = true
			_ = request.Reply(true, nil)
			go func() {
				_, _ = io.Copy(terminal, channel)
				stop()
			}()
			outputDone := make(chan struct{})
			go func() {
				_, _ = io.Copy(channel, terminal)
				close(outputDone)
			}()
			go func() {
				err := command.Wait()
				status := uint32(0)
				if err != nil {
					status = 1
					if !errors.Is(err, context.Canceled) {
						log.Printf("PowerShell exited: %v", err)
					}
				}
				select {
				case <-outputDone:
				case <-time.After(500 * time.Millisecond):
				}
				stop()
				select {
				case <-outputDone:
				case <-time.After(2 * time.Second):
				}
				_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
				_ = channel.Close()
			}()
		default:
			_ = request.Reply(false, nil)
		}
	}
}

func parsePtyRequest(payload []byte) (windowSize, bool) {
	if len(payload) < 4 {
		return windowSize{}, false
	}
	termLength := int(binary.BigEndian.Uint32(payload[:4]))
	if termLength > len(payload)-24 || termLength < 0 {
		return windowSize{}, false
	}
	return parseDimensions(payload[4+termLength:])
}

func parseWindowChange(payload []byte) (windowSize, bool) {
	return parseDimensions(payload)
}

func parseDimensions(payload []byte) (windowSize, bool) {
	if len(payload) < 8 {
		return windowSize{}, false
	}
	width := binary.BigEndian.Uint32(payload[:4])
	height := binary.BigEndian.Uint32(payload[4:8])
	if width < 1 || width > 1000 || height < 1 || height > 1000 {
		return windowSize{}, false
	}
	return windowSize{int(width), int(height)}, true
}
