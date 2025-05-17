package uploader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gconfig "github.com/navaneethkn/cronocam/internal/config"
)

type Config struct {
	ChunkSize         int64
	MaxRetries        int
	RequestsPerSecond int
	MaxBurst          int
}

type Uploader struct {
	client      *http.Client
	config      Config
	supported   map[string]bool
	rateLimiter *RateLimiter
}

func New(client *http.Client, config Config) (*Uploader, error) {
	// Get supported formats from config
	supported := gconfig.GetSupportedFormats()

	return &Uploader{
		client:      client,
		config:      config,
		supported:   supported,
		rateLimiter: NewRateLimiter(config.RequestsPerSecond, config.MaxBurst),
	}, nil
}

func (u *Uploader) IsSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return u.supported[ext]
}

func (u *Uploader) CalculateFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (u *Uploader) UploadFile(ctx context.Context, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to open file: %v", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("unable to get file info: %v", err)
	}

	// Start resumable upload session
	uploadURL, err := u.startResumableUpload(ctx, filePath, fileInfo.Size())
	if err != nil {
		return "", fmt.Errorf("unable to start upload: %v", err)
	}

	// Upload file in chunks
	uploadToken, err := u.uploadChunks(ctx, file, uploadURL, fileInfo.Size())
	if err != nil {
		return "", fmt.Errorf("chunk upload failed: %v", err)
	}

	// Create media item
	item, err := u.createMediaItem(ctx, uploadToken, filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create media item: %v", err)
	}

	return item.ID, nil
}

func (u *Uploader) startResumableUpload(ctx context.Context, filePath string, size int64) (string, error) {
	url := "https://photoslibrary.googleapis.com/v1/uploads"
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}

	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "image/jpeg" // fallback for unknown extensions
	}

	req.Header.Set("X-Goog-Upload-Protocol", "resumable")
	req.Header.Set("X-Goog-Upload-Command", "start")
	req.Header.Set("X-Goog-Upload-Content-Type", contentType)
	req.Header.Set("X-Goog-Upload-Raw-Size", fmt.Sprintf("%d", size))
	req.Header.Set("Content-Length", "0")

	resp, err := u.client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to start upload, status: %d, body: %s", resp.StatusCode, string(body))
	}

	uploadURL := resp.Header.Get("X-Goog-Upload-URL")
	if uploadURL == "" {
		return "", fmt.Errorf("no upload URL in response")
	}

	return uploadURL, nil
}

func (u *Uploader) uploadChunks(ctx context.Context, file *os.File, uploadURL string, totalSize int64) (string, error) {
	buffer := make([]byte, u.config.ChunkSize)
	offset := int64(0)
	var uploadToken string

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}

		chunk := buffer[:n]
		isLast := offset+int64(n) >= totalSize
		cmd := "upload"
		if isLast {
			cmd = "upload, finalize"
		}

		req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, bytes.NewReader(chunk))
		if err != nil {
			return "", err
		}

		req.Header.Set("X-Goog-Upload-Command", cmd)
		req.Header.Set("X-Goog-Upload-Offset", fmt.Sprintf("%d", offset))
		req.Header.Set("Content-Length", fmt.Sprintf("%d", n))

		resp, err := u.client.Do(req)
		if err != nil {
			return "", err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("failed to read response: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("chunk upload failed, status: %d, body: %s", resp.StatusCode, string(body))
		}

		if isLast {
			// Get upload token from the last response
			uploadToken = string(body)
			break
		}

		offset += int64(n)
	}

	if uploadToken == "" {
		return "", fmt.Errorf("no upload token received")
	}

	return uploadToken, nil
}

type mediaItem struct {
	ID string `json:"id"`
}

type mediaItemResult struct {
	Status struct {
		Message string `json:"message"`
	} `json:"status"`
	MediaItem mediaItem `json:"mediaItem"`
}

type batchCreateResponse struct {
	NewMediaItemResults []mediaItemResult `json:"newMediaItemResults"`
}

func (u *Uploader) createMediaItem(ctx context.Context, uploadToken, filename string) (*mediaItem, error) {
	url := "https://photoslibrary.googleapis.com/v1/mediaItems:batchCreate"

	reqBody := map[string]interface{}{
		"newMediaItems": []map[string]interface{}{
			{
				"simpleMediaItem": map[string]string{
					"uploadToken": uploadToken,
				},
				"description": filename,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= u.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			waitTime := time.Duration(attempt) * time.Second * 2
			time.Sleep(waitTime)
		}

		// Wait for rate limiter
		if err := u.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter wait failed: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		resp, err := u.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %v", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limit exceeded (429), body: %s", string(body))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("failed to create media item, status: %d, body: %s", resp.StatusCode, string(body))
			if resp.StatusCode < 500 { // Don't retry 4xx errors except 429
				return nil, lastErr
			}
			continue
		}

		var result batchCreateResponse
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
			lastErr = fmt.Errorf("failed to decode response: %v", err)
			continue
		}

		if len(result.NewMediaItemResults) == 0 {
			lastErr = fmt.Errorf("no media items created")
			continue
		}

		if result.NewMediaItemResults[0].Status.Message != "Success" {
			lastErr = fmt.Errorf("failed to create media item: %s", result.NewMediaItemResults[0].Status.Message)
			continue
		}

		return &result.NewMediaItemResults[0].MediaItem, nil
	}

	return nil, fmt.Errorf("all retries failed, last error: %v", lastErr)
}
