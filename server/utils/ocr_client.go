package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OCRRunner extracts text and per-word boxes from raw image bytes. It is the
// abstraction the broker depends on, so the OCR engine can be swapped out or
// run out-of-process without the broker caring.
type OCRRunner interface {
	Extract(ctx context.Context, imageData []byte) (string, []WordBox, error)
}

// HTTPOCRClient talks to the sandboxed ocr-service over HTTP. The service is
// expected to live in an isolated, low-privilege container so that exploits in
// libtesseract / libleptonica / image decoders can't reach this process's
// secrets or database.
type HTTPOCRClient struct {
	baseURL string
	http    *http.Client
}

func NewHTTPOCRClient(baseURL string, timeout time.Duration) *HTTPOCRClient {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &HTTPOCRClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

type ocrResponse struct {
	Text  string    `json:"text"`
	Words []WordBox `json:"words"`
}

func (c *HTTPOCRClient) Extract(ctx context.Context, imageData []byte) (string, []WordBox, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ocr", bytes.NewReader(imageData))
	if err != nil {
		return "", nil, fmt.Errorf("build ocr request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("ocr service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", nil, fmt.Errorf("ocr service returned %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var r ocrResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", nil, fmt.Errorf("decode ocr response: %w", err)
	}
	return r.Text, r.Words, nil
}
