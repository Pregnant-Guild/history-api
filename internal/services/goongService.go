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
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type GoongService interface {
	ProxyRequest(ctx context.Context, method string, targetURL string, headers map[string]string, reqBody []byte, apiKeyName string) (int, map[string]string, []byte, error)
}

type CacheEntry struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

type goongService struct {
	redis      cache.Cache
	httpClient *http.Client
}

func NewGoongService(redis cache.Cache) GoongService {
	return &goongService{
		redis: redis,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (s *goongService) ProxyRequest(ctx context.Context, method string, targetURL string, headers map[string]string, reqBody []byte, apiKeyName string) (int, map[string]string, []byte, error) {
	apiKey, err := config.GetConfig(apiKeyName)
	if err != nil {
		return 500, nil, nil, fmt.Errorf("%s is not configured", apiKeyName)
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

	cacheKey := cache.Key("goong_proxy_v2", finalURL)

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
		if lowerK == "host" || lowerK == "connection" || lowerK == "accept-encoding" || lowerK == "keep-alive" || lowerK == "te" || lowerK == "transfer-encoding" || lowerK == "upgrade" {
			continue
		}
		if strings.HasPrefix(lowerK, "cf-") {
			continue
		}
		if strings.HasPrefix(lowerK, "x-forwarded-") || lowerK == "x-real-ip" || lowerK == "true-client-ip" || lowerK == "cdn-loop" {
			continue
		}
		if lowerK == "cookie" || lowerK == "authorization" || lowerK == "x-api-key" || lowerK == "x-csrf-token" {
			continue
		}
		if strings.HasPrefix(lowerK, "sec-") || lowerK == "user-agent" || lowerK == "accept-language" || lowerK == "priority" || lowerK == "purpose" || lowerK == "dnt" {
			continue
		}
		req.Header.Set(k, v)
	}

	var resp *http.Response
	var lastErr error
	var respBody []byte

	for attempt := 1; attempt <= 3; attempt++ {
		if ctx.Err() != nil {
			lastErr = ctx.Err()
			break
		}

		attemptReq := req.Clone(ctx)
		if len(reqBody) > 0 {
			attemptReq.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		resp, err = s.httpClient.Do(attemptReq)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				break
			}
			time.Sleep(time.Duration(attempt*150) * time.Millisecond)
			continue
		}

		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt*150) * time.Millisecond)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			break
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("target returned status code %d: %s", resp.StatusCode, string(respBody))
			time.Sleep(time.Duration(attempt*150) * time.Millisecond)
			continue
		}

		break
	}

	if resp == nil || (resp.StatusCode != http.StatusOK && lastErr != nil) {
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		if ctx.Err() != nil {
			return 499, nil, nil, ctx.Err()
		}
		log.Error().
			Err(lastErr).
			Int("status_code", statusCode).
			Str("url", finalURL).
			Str("method", method).
			Msg("Goong Map proxy request failed after retries")
		return statusCode, nil, nil, fmt.Errorf("failed to fetch from target: %w", lastErr)
	}

	if resp.StatusCode != http.StatusOK {
		log.Warn().
			Int("status_code", resp.StatusCode).
			Str("url", finalURL).
			Str("method", method).
			Str("resp_body", string(respBody)).
			Msg("Goong Map proxy request returned non-200 status code")
	}

	respHeaders := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		lowerK := strings.ToLower(k)
		// Skip hop-by-hop/transport headers in response
		if lowerK == "connection" || lowerK == "keep-alive" || lowerK == "transfer-encoding" || lowerK == "upgrade" {
			continue
		}
		// Skip Cloudflare headers from Goong to prevent clashing with our own server
		if strings.HasPrefix(lowerK, "cf-") || lowerK == "server" || lowerK == "date" || lowerK == "alt-svc" {
			continue
		}
		// Skip CORS headers (handled by our own Fiber CORS middleware)
		if strings.HasPrefix(lowerK, "access-control-") {
			continue
		}
		// Skip content encoding/length (handled by Fiber / Go client automatically)
		if lowerK == "content-encoding" || lowerK == "content-length" {
			continue
		}
		respHeaders[k] = v[0]
	}

	contentType := resp.Header.Get("Content-Type")
	isText := strings.Contains(contentType, "json") ||
		strings.Contains(contentType, "text") ||
		strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "xml")

	if isText && len(respBody) > 0 {
		pattern := fmt.Sprintf(`(?i)([?&])api_key=%s(&?)`, regexp.QuoteMeta(apiKey))
		if re, err := regexp.Compile(pattern); err == nil {
			respBodyStr := string(respBody)
			cleanedStr := re.ReplaceAllStringFunc(respBodyStr, func(match string) string {
				groups := re.FindStringSubmatch(match)
				if len(groups) > 2 && groups[2] == "&" {
					return groups[1]
				}
				return ""
			})
			respBody = []byte(cleanedStr)
		}
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
