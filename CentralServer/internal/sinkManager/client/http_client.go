package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"central_server/internal/sinkManager/domain"
)

// HTTPSinkClient implements SinkClient interface for HTTP communication
type HTTPSinkClient struct {
	httpClient *http.Client
}

func NewHTTPSinkClient(timeout time.Duration) *HTTPSinkClient {
	return &HTTPSinkClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SendHeartbeat sends a heartbeat request to the sink and returns the response
func (c *HTTPSinkClient) SendHeartbeat(ctx context.Context, sink *domain.Sink) (*domain.HeartbeatRequest, error) {
	url := fmt.Sprintf("%s/heartbeat", sink.Endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("heartbeat returned status %d", resp.StatusCode)
	}

	var heartbeatResp domain.HeartbeatRequest
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		return nil, fmt.Errorf("failed to decode heartbeat response: %w", err)
	}

	return &heartbeatResp, nil
}

// DeliverTask sends a task to the sink for execution
func (c *HTTPSinkClient) DeliverTask(ctx context.Context, sink *domain.Sink, task *domain.TaskDeliveryRequest) (*domain.TaskDeliveryResponse, error) {
	url := fmt.Sprintf("%s/execute", sink.Endpoint)

	body, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create delivery request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("task delivery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("task delivery returned status %d", resp.StatusCode)
	}

	var deliveryResp domain.TaskDeliveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&deliveryResp); err != nil {
		return nil, fmt.Errorf("failed to decode delivery response: %w", err)
	}

	return &deliveryResp, nil
}
