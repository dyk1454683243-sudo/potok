package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	serverURL string
	apiKey    string
}

func New(serverURL, apiKey string) *Client {
	return &Client{serverURL: serverURL, apiKey: apiKey}
}

func (c *Client) Request(method, path string) (*http.Response, error) {
	return c.RequestBody(method, path, "", nil)
}

func (c *Client) RequestBody(method, path, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.serverURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return response, nil
}

func (c *Client) RequestJSON(method, path string, out any) (*http.Response, error) {
	response, err := c.Request(method, path)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return response, nil
	}

	if out != nil {
		if err := json.NewDecoder(response.Body).Decode(out); err != nil {
			return response, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return response, nil
}

func (c *Client) RequestJSONBody(method, path string, in, out any) (*http.Response, error) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(in); err != nil {
		return nil, fmt.Errorf("failed to encode request body: %w", err)
	}

	response, err := c.RequestBody(method, path, "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return response, nil
	}

	if out != nil {
		if err := json.NewDecoder(response.Body).Decode(out); err != nil {
			return response, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return response, nil
}

func ReadMessage(response *http.Response) string {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return response.Status
	}
	return string(bytes.TrimSpace(body))
}
