package modelrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ChatResponse struct {
	Model   string  `json:"model"`
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

func NewClient(baseURL, model string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	model = strings.TrimSpace(model)

	if baseURL == "" {
		return nil, fmt.Errorf("Ollama base URL is required")
	}

	if model == "" {
		return nil, fmt.Errorf("Ollama model name is required")
	}

	return &Client{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 3 * time.Minute,
		},
	}, nil
}

func (c *Client) Chat(
	ctx context.Context,
	messages []Message,
) (*ChatResponse, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}

	requestBody := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/chat",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contact Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf(
			"Ollama returned HTTP %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	var result ChatResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Ollama response: %w", err)
	}

	if strings.TrimSpace(result.Message.Content) == "" {
		return nil, fmt.Errorf("Ollama returned an empty response")
	}

	return &result, nil
}

func (c *Client) Model() string {
	return c.model
}
