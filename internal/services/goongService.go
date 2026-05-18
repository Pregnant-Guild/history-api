package services

import (
	"bytes"
	"context"
	"fmt"
	"history-api/pkg/cache"
	"history-api/pkg/config"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GoongService interface {
	ProxyRequest(ctx context.Context, method string, targetURL string, headers map[string]string, reqBody []byte) (int, map[string]string, []byte, error)
}

type CacheEntry struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

type goongService struct {
	redis cache.Cache
}

func NewGoongService(redis cache.Cache) GoongService {
	return &goongService{
		redis: redis,
	}
}

func (s *goongService) ProxyRequest(ctx context.Context, method string, targetURL string, headers map[string]string, reqBody []byte) (int, map[string]string, []byte, error) {
	apiKey, err := config.GetConfig("GOONG_API_KEY")
	if err != nil {
		return 500, nil, nil, fmt.Errorf("GOONG_API_KEY is not configured")
	}

	fullTargetURL := targetURL
	if !strings.HasPrefix(fullTargetURL, "http://") && !strings.HasPrefix(fullTargetURL, "https://") {
		fullTargetURL = "https://" + fullTargetURL
	}

	parsedUrl, err := url.Parse(fullTargetURL)
	if err != nil || parsedUrl.Host == "" {
		return 400, nil, nil, fmt.Errorf("invalid target URL")
	}

	if !strings.HasSuffix(parsedUrl.Host, "goong.io") {
		return 403, nil, nil, fmt.Errorf("only goong.io domains are allowed")
	}

	q := parsedUrl.Query()
	q.Set("api_key", apiKey)
	parsedUrl.RawQuery = q.Encode()
	finalURL := parsedUrl.String()

	cacheKey := fmt.Sprintf("goong_proxy:%s", finalURL)

	if method == http.MethodGet {
		var entry CacheEntry
		if err := s.redis.Get(ctx, cacheKey, &entry); err == nil && len(entry.Body) > 0 {
			return entry.StatusCode, entry.Headers, entry.Body, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, finalURL, nil)
	if err != nil {
		return 500, nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if len(reqBody) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	for k, v := range headers {
		lowerK := strings.ToLower(k)
		if lowerK == "host" || lowerK == "connection" || lowerK == "accept-encoding" {
			continue
		}
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 500, nil, nil, fmt.Errorf("failed to fetch from target: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 500, nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		respHeaders[k] = v[0]
	}

	if method == http.MethodGet && resp.StatusCode == http.StatusOK {
		entry := CacheEntry{
			StatusCode: resp.StatusCode,
			Headers:    respHeaders,
			Body:       respBody,
		}
		_ = s.redis.Set(ctx, cacheKey, entry, 24*time.Hour)
	}

	return resp.StatusCode, respHeaders, respBody, nil
}
