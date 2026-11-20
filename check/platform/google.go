package platform

import (
	"context"
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
	url := strings.TrimSpace(config.GlobalConfig.AliveCheckURL)
	if url == "" {
		url = "https://gstatic.com/generate_204"
	}
	return checkAliveEndpoint(httpClient, ctx, url)
}


// checkAliveEndpoint 只判断配置的保活地址是否能通过节点收到 HTTP 响应。
// 任意 HTTP 状态码都视为链路可达，节点是否保留再由延迟等筛选条件决定。
func checkAliveEndpoint(httpClient *http.Client, ctx context.Context, url string) (bool, error) {
\treq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
\tif err != nil {
\t\treturn false, err
\t}

\treq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
\treq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
\treq.Header.Set("Accept-Language", "en-US,en;q=0.5")
\t// 请求少量内容即可确认链路，避免下载完整网页或图标。
\treq.Header.Set("Range", "bytes=0-0")
\treq.Close = true

\tresp, err := httpClient.Do(req)
\tif err != nil {
\t\tslog.Debug(err.Error())
\t\treturn false, err
\t}
\tdefer resp.Body.Close()

\treturn true, nil
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
