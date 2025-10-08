//go:build unix

package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gurisko/takl/internal/limits"
	"github.com/gurisko/takl/internal/paths"
)

const (
	// MaxJSONPayloadSize is the maximum size for JSON payloads in the API (1MB)
	// This applies to both client requests and server responses
	// Deprecated: Use limits.JSON instead
	MaxJSONPayloadSize = limits.JSON
)

type Client struct {
	http       *http.Client
	baseURL    string
	socketPath string
}

func New() *Client {
	socketPath := paths.DefaultSocketPath()
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &Client{
		http:       &http.Client{Transport: tr}, // no Timeout; use ctx per-request
		baseURL:    "http://unix",
		socketPath: socketPath,
	}
}

type APIError struct {
	StatusCode int
	Body       []byte
	Message    string // parsed from {"error": "..."} if present
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("api error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api error %d: %s", e.StatusCode, string(e.Body))
}

func decodeAPIError(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, limits.ErrorBody))
	var m struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(b, &m)
	return &APIError{StatusCode: resp.StatusCode, Body: b, Message: m.Error}
}

func (c *Client) GetJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return c.wrapConnErr(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeAPIError(resp)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, limits.JSON)).Decode(out)
}

func (c *Client) PostJSON(ctx context.Context, path string, in any, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return c.wrapConnErr(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 && resp.StatusCode != http.StatusCreated {
		return decodeAPIError(resp)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, limits.JSON)).Decode(out)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return c.wrapConnErr(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 && resp.StatusCode != http.StatusNoContent {
		return decodeAPIError(resp)
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

func IsNotFound(err error) bool {
	var api *APIError
	return errors.As(err, &api) && api.StatusCode == http.StatusNotFound
}

// Friendly hint when the daemon isn't running / socket missing.
func (c *Client) wrapConnErr(err error) error {
	// best-effort heuristics without importing x/sys
	if strings.Contains(err.Error(), "connect: no such file or directory") ||
		strings.Contains(err.Error(), "unknown network unix") ||
		strings.Contains(err.Error(), "connection refused") {
		return fmt.Errorf("cannot connect to takl daemon at %s; is it running? try `takl daemon start` (%w)", c.socketPath, err)
	}
	return err
}
