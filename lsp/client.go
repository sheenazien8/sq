package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sheenazien8/sq/logger"
)

// Message represents a JSON-RPC message
type Message struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Client represents an LSP client
type Client struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	responses  chan Message
	requests   map[string]chan Message
	requestMux sync.RWMutex
	nextID     int
	running    bool
	command    string
	args       []string
}

// NewClient creates a new LSP client with the given command and args
func NewClient(command string, args ...string) *Client {
	return &Client{
		requests:  make(map[string]chan Message),
		responses: make(chan Message, 100),
		nextID:    1,
		command:   command,
		args:      args,
	}
}

// NewClientWithConfig creates a new LSP client configured for sqls with a config file
func NewClientWithConfig(configPath string) *Client {
	args := []string{"-config", configPath}
	return NewClient("sqls", args...)
}

// Start starts the LSP server process
func (c *Client) Start() error {
	logger.Debug("Starting LSP server", map[string]any{
		"command": c.command,
		"args":    c.args,
	})

	c.cmd = exec.Command(c.command, c.args...)

	// Redirect stderr to /dev/null to prevent sqls output from interfering with TUI
	if nullFile, err := os.OpenFile("/dev/null", os.O_WRONLY, 0); err == nil {
		c.cmd.Stderr = nullFile
	} else {
		// Fallback to os.Stderr if /dev/null can't be opened
		c.cmd.Stderr = os.Stderr
	}

	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	c.stdout = bufio.NewReader(stdout)

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start LSP server: %w", err)
	}

	c.running = true

	// Start response reader
	go c.readResponses()

	logger.Debug("LSP server started successfully", nil)
	return nil
}

// Stop stops the LSP server process
func (c *Client) Stop() error {
	if !c.running {
		return nil
	}

	c.running = false

	if c.stdin != nil {
		c.stdin.Close()
	}

	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			logger.Debug("Error killing LSP process", map[string]any{"error": err.Error()})
		}
		c.cmd.Wait()
	}

	return nil
}

// SendRequest sends a request and waits for response
func (c *Client) SendRequest(method string, params interface{}) (Message, error) {
	id := c.nextID
	c.nextID++

	req := Message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	responseChan := make(chan Message, 1)
	c.requestMux.Lock()
	c.requests[fmt.Sprintf("%v", id)] = responseChan
	c.requestMux.Unlock()

	if err := c.sendMessage(req); err != nil {
		c.requestMux.Lock()
		delete(c.requests, fmt.Sprintf("%v", id))
		c.requestMux.Unlock()
		return Message{}, err
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(10 * time.Second):
		c.requestMux.Lock()
		delete(c.requests, fmt.Sprintf("%v", id))
		c.requestMux.Unlock()
		return Message{}, fmt.Errorf("request timeout")
	}
}

// SendNotification sends a notification (no response expected)
func (c *Client) SendNotification(method string, params interface{}) error {
	req := Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	return c.sendMessage(req)
}

// sendMessage sends a message to the LSP server
func (c *Client) sendMessage(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// LSP uses Content-Length header
	content := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)

	logger.Debug("Sending LSP message", map[string]any{
		"method": msg.Method,
		"id":     msg.ID,
		"length": len(content),
	})

	_, err = c.stdin.Write([]byte(content))
	return err
}

// readResponses reads responses from the LSP server using proper LSP protocol parsing
func (c *Client) readResponses() {
	for c.running {
		// Read headers until we get an empty line
		var contentLength int
		for {
			line, err := c.stdout.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					logger.Debug("Error reading LSP header", map[string]any{"error": err.Error()})
				}
				return
			}

			line = strings.TrimSpace(line)

			// Empty line marks end of headers
			if line == "" {
				break
			}

			// Parse Content-Length header
			if strings.HasPrefix(line, "Content-Length:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					length, err := strconv.Atoi(strings.TrimSpace(parts[1]))
					if err != nil {
						logger.Debug("Invalid Content-Length value", map[string]any{"value": parts[1], "error": err.Error()})
						continue
					}
					contentLength = length
				}
			}
		}

		if contentLength == 0 {
			logger.Debug("No Content-Length header found", nil)
			continue
		}

		// Read exactly contentLength bytes
		content := make([]byte, contentLength)
		n, err := io.ReadFull(c.stdout, content)
		if err != nil {
			logger.Debug("Error reading LSP content", map[string]any{
				"error":    err.Error(),
				"expected": contentLength,
				"read":     n,
			})
			continue
		}

		var msg Message
		if err := json.Unmarshal(content, &msg); err != nil {
			logger.Debug("Failed to unmarshal LSP message", map[string]any{
				"content": string(content),
				"error":   err.Error(),
			})
			continue
		}

		logger.Debug("Received LSP message", map[string]any{
			"method": msg.Method,
			"id":     msg.ID,
		})

		// Handle response
		if msg.ID != nil {
			c.requestMux.RLock()
			ch, exists := c.requests[fmt.Sprintf("%v", msg.ID)]
			c.requestMux.RUnlock()

			if exists {
				select {
				case ch <- msg:
				default:
					logger.Debug("Response channel full, dropping message", map[string]any{"id": msg.ID})
				}

				c.requestMux.Lock()
				delete(c.requests, fmt.Sprintf("%v", msg.ID))
				c.requestMux.Unlock()
			} else {
				logger.Debug("No waiting request for response", map[string]any{"id": msg.ID})
			}
		} else {
			// Handle notification
			select {
			case c.responses <- msg:
			default:
				logger.Debug("Response channel full, dropping notification", nil)
			}
		}
	}
}

// Initialize sends the initialize request
func (c *Client) Initialize(rootURI string, capabilities map[string]interface{}) (Message, error) {
	params := map[string]interface{}{
		"processId":    nil,
		"rootUri":      rootURI,
		"capabilities": capabilities,
	}

	return c.SendRequest("initialize", params)
}

// Initialized sends the initialized notification
func (c *Client) Initialized() error {
	return c.SendNotification("initialized", map[string]interface{}{})
}

// DidOpen sends textDocument/didOpen notification
func (c *Client) DidOpen(uri, languageId, text string) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": languageId,
			"version":    1,
			"text":       text,
		},
	}

	return c.SendNotification("textDocument/didOpen", params)
}

// DidChange sends textDocument/didChange notification
func (c *Client) DidChange(uri, text string, version int) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": []map[string]interface{}{
			{
				"text": text,
			},
		},
	}

	return c.SendNotification("textDocument/didChange", params)
}

// Completion sends textDocument/completion request
func (c *Client) Completion(uri string, line, character int) (Message, error) {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"position": map[string]interface{}{
			"line":      line,
			"character": character,
		},
	}

	return c.SendRequest("textDocument/completion", params)
}

// Hover sends textDocument/hover request
func (c *Client) Hover(uri string, line, character int) (Message, error) {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"position": map[string]interface{}{
			"line":      line,
			"character": character,
		},
	}

	return c.SendRequest("textDocument/hover", params)
}
