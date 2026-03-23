package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

const (
	defaultHTTPTimeout    = 15 * time.Second
	blockingRequestBuffer = 5 * time.Second
)

type Client struct {
	baseURL string
	http    *http.Client
}

type OutResponse struct {
	OK    bool   `json:"ok"`
	Space string `json:"space"`
	Arity int    `json:"arity"`
}

type ReadResponse struct {
	OK       bool           `json:"ok"`
	Tuple    types.Tuple    `json:"tuple"`
	Bindings map[string]any `json:"bindings"`
}

type HealthResponse struct {
	OK bool `json:"ok"`
}

type errorEnvelope struct {
	OK    bool `json:"ok"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func New(baseURL string) *Client {
	return NewWithTimeout(baseURL, defaultHTTPTimeout)
}

func NewWithTimeout(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

func TimeoutForWaitMS(waitMS int64) time.Duration {
	if waitMS <= 0 {
		return defaultHTTPTimeout
	}

	timeout := time.Duration(waitMS)*time.Millisecond + blockingRequestBuffer
	if timeout < defaultHTTPTimeout {
		return defaultHTTPTimeout
	}
	return timeout
}

func (c *Client) Out(ctx context.Context, space string, tuple types.Tuple) (*OutResponse, error) {
	var response OutResponse
	err := c.post(ctx, "/v1/spaces/"+space+"/out", map[string]any{
		"tuple": tuple,
	}, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) Rd(ctx context.Context, space string, template types.Template, waitMS int64) (*ReadResponse, error) {
	return c.read(ctx, "/v1/spaces/"+space+"/rd", template, waitMS)
}

func (c *Client) In(ctx context.Context, space string, template types.Template, waitMS int64) (*ReadResponse, error) {
	return c.read(ctx, "/v1/spaces/"+space+"/in", template, waitMS)
}

func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return nil, fmt.Errorf("build health request: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform health request: %w", err)
	}
	defer res.Body.Close()

	var response HealthResponse
	if err := decodeResponse(res, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) read(ctx context.Context, path string, template types.Template, waitMS int64) (*ReadResponse, error) {
	var response ReadResponse
	err := c.post(ctx, path, map[string]any{
		"template": template,
		"wait_ms":  waitMS,
	}, &response)
	if err != nil {
		return nil, err
	}
	normalizedTuple, err := types.NormalizeTuple(response.Tuple)
	if err != nil {
		return nil, fmt.Errorf("normalize response tuple: %w", err)
	}
	response.Tuple = normalizedTuple
	return &response, nil
}

func (c *Client) post(ctx context.Context, path string, requestBody any, dst any) error {
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer res.Body.Close()

	return decodeResponse(res, dst)
}

func decodeResponse(res *http.Response, dst any) error {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if res.StatusCode >= 400 {
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		var envelope errorEnvelope
		if err := decoder.Decode(&envelope); err == nil {
			if envelope.Error.Message != "" {
				return fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
			}
		}

		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("request failed with status %d", res.StatusCode)
		}
		return fmt.Errorf("request failed with status %d: %s", res.StatusCode, message)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
