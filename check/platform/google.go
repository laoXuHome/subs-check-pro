package platform

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sinspired/subs-check-pro/v3/config"
)

func CheckGoogle(httpClient *http.Client, ctx context.Context) (bool, error) {
	if success, err := checkGoogleEndpoint(httpClient, ctx, "https://gstatic.com/generate_204", 204); err == nil && success {
		return checkGoogleEndpoint(httpClient, ctx, "https://www.google.com/generate_204", 204)
	}
	return false, nil
}

func CheckGstatic(httpClient *http.Client, ctx context.Context) (bool, error) {
	urls := config.GlobalConfig.AliveCheckURLs
	if len(urls) == 0 {
		url := strings.TrimSpace(config.GlobalConfig.AliveCheckURL)
		if url == "" {
			url = "https://gstatic.com/generate_204"
		}
		urls = []string{url}
	}

	var lastErr error
	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		alive, err := checkAliveEndpoint(httpClient, ctx, url)
		if alive {
			return true, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return false, lastErr
	}
	return false, errors.New("未配置有效的节点保活地址")
}

// checkAliveEndpoint 只判断配置的保活地址是否能通过节点收到 HTTP 响应。
// 任意 HTTP 状态码都视为链路可达；节点是否进入后续阶段由其他筛选条件决定。
func checkAliveEndpoint(httpClient *http.Client, ctx context.Context, url string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// 请求少量内容即可确认链路，避免下载完整网页或图标。
	req.Header.Set("Range", "bytes=0-0")
	req.Close = true

	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Debug(err.Error())
		return false, err
	}
	defer resp.Body.Close()

	return true, nil
}

// checkGoogleEndpoint 使用 HEAD 方法检查 URL 的状态码。
func checkGoogleEndpoint(httpClient *http.Client, ctx context.Context, url string, statusCode int) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// 释放连接池和内存
	req.Close = true

	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Debug(err.Error())
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == statusCode, nil
}
