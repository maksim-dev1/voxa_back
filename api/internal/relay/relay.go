// Package relay is voxa-api's client for voxa-storage-relay on nexus.
// It streams the uploaded audio through without buffering the whole file
// in memory or on Pi's disk — api is stateless by design.
package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{}}
}

type Upload struct {
	UserID      string
	Filename    string
	ContentType string
	Body        io.Reader
}

type ingestResponse struct {
	JobID string `json:"job_id"`
}

// UploadRecording streams the file to storage-relay's /ingest endpoint via
// a piped multipart body, so the request body is never fully materialized
// in memory regardless of recording length.
func (c *Client) UploadRecording(ctx context.Context, u Upload) (string, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		if err := writer.WriteField("user_id", u.UserID); err != nil {
			pw.CloseWithError(err)
			return
		}
		part, err := writer.CreateFormFile("audio", u.Filename)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, u.Body); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ingest", pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("storage-relay returned %d: %s", resp.StatusCode, body)
	}

	var out ingestResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.JobID, nil
}
