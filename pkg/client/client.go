package client

import (
	"bytes"
	"context"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type apiError struct {
	Error string `json:"error"`
}

func New(baseURL string, httpClient *http.Client) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) Enqueue(ctx context.Context, req EnqueueRequest) (*Job, error) {
	var job Job
	if err := c.do(ctx, http.MethodPost, "/jobs", req, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *Client) Job(ctx context.Context, jobID string) (*Job, error) {
	var job Job
	if err := c.do(ctx, http.MethodGet, "/jobs/"+jobID, nil, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *Client) Queues(ctx context.Context) (*QueuesResponse, error) {
	var resp QueuesResponse
	if err := c.do(ctx, http.MethodGet, "/queues", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) QueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	var stats QueueStats
	if err := c.do(ctx, http.MethodGet, "/queues/"+queueName+"/stats", nil, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (c *Client) DLQ(ctx context.Context) ([]Job, error) {
	var jobs []Job
	if err := c.do(ctx, http.MethodGet, "/dlq", nil, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (c *Client) RequeueDLQ(ctx context.Context, jobID string, req RequeueRequest) (*Job, error) {
	var job Job
	if err := c.do(ctx, http.MethodPost, "/dlq/"+jobID+"/requeue", req, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, dst any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr apiError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil || apiErr.Error == "" {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}
		return stdErrors.New(apiErr.Error)
	}

	if dst == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
