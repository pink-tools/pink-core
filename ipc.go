package core

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pink-tools/pink-core/log"
)

// startIPCListener starts TCP listener for graceful shutdown and custom commands
// Returns cleanup function and error
func startIPCListener(name string, cancel context.CancelFunc, handler func(string) string) (func(), error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	// Get assigned port
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Write port to file
	portFile := portFilePath(name)
	if err := os.MkdirAll(filepath.Dir(portFile), 0755); err != nil {
		listener.Close()
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0644); err != nil {
		listener.Close()
		return nil, fmt.Errorf("write port file: %w", err)
	}

	// Accept connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}
			go handleIPCConnection(conn, cancel, handler)
		}
	}()

	cleanup := func() {
		listener.Close()
		os.Remove(portFile)
	}

	return cleanup, nil
}

func handleIPCConnection(conn net.Conn, cancel context.CancelFunc, handler func(string) string) {
	defer conn.Close()

	reader := bufio.NewReaderSize(conn, 65536)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	cmd := strings.TrimSpace(line)
	switch cmd {
	case "STOP":
		log.Info(context.Background(), "received IPC STOP command")
		conn.Write([]byte("OK\n"))
		cancel()
	case "PING":
		conn.Write([]byte("PONG\n"))
	default:
		if handler != nil {
			response := handler(cmd)
			conn.Write([]byte(response + "\n"))
		} else {
			conn.Write([]byte("UNKNOWN\n"))
		}
	}
}

// SendStop sends STOP command via IPC
func SendStop(name string) error {
	port, err := readPort(name)
	if err != nil {
		return fmt.Errorf("not running: %w", err)
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	conn.Write([]byte("STOP\n"))

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if strings.TrimSpace(response) != "OK" {
		return fmt.Errorf("unexpected response: %s", response)
	}

	return nil
}

// SendCommand sends a command via IPC and returns response
func SendCommand(name, cmd string) (string, error) {
	port, err := readPort(name)
	if err != nil {
		return "", fmt.Errorf("not running: %w", err)
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	conn.Write([]byte(cmd + "\n"))

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// IsOrchestratorRunning checks if pink-orchestrator is running
func IsOrchestratorRunning() bool {
	return IsRunning("pink-orchestrator")
}

// ShowDialog sends a dialog request to orchestrator and returns user choice
// Returns "confirm", "cancel", or error message
// Falls back to empty string if orchestrator not running
func ShowDialog(dialogJSON string) (string, error) {
	if !IsOrchestratorRunning() {
		return "", fmt.Errorf("orchestrator not running")
	}
	return SendCommand("pink-orchestrator", "dialog:"+dialogJSON)
}

// IsRunning checks if service is running via IPC
func IsRunning(name string) bool {
	port, err := readPort(name)
	if err != nil {
		return false
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	defer conn.Close()

	conn.Write([]byte("PING\n"))

	reader := bufio.NewReader(conn)
	response, _ := reader.ReadString('\n')
	return strings.TrimSpace(response) == "PONG"
}

func readPort(name string) (int, error) {
	data, err := os.ReadFile(portFilePath(name))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func portFilePath(name string) string {
	return filepath.Join(DataDir(name), name+".port")
}
