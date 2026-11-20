package proxies

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sinspired/subs-check-pro/v3/config"
	"github.com/sinspired/subs-check-pro/v3/proxy/parse"
	"github.com/sinspired/subs-check-pro/v3/utils"
)

var subscriptionCacheLocks sync.Map

// FetchSubsDataWithPolicy applies the configured refresh policy to subscription
// bodies. Cache keys are hashes so subscription URLs and credentials are not
// exposed in cache filenames.
func FetchSubsDataWithPolicy(rawURL string) ([]byte, error) {
	urlKey := strings.TrimSpace(parse.CleanURL(rawURL))
	if urlKey == "" {
		return nil, errors.New("订阅 URL 为空")
	}
	urlKey = utils.NormalizeGitHubRawURL(urlKey)
	urlKey = parse.EnsureScheme(urlKey)

	policy := strings.ToLower(strings.TrimSpace(config.GlobalConfig.SubFetchPolicy))
	if policy == "" {
		policy = "always"
	}
	if policy != "always" && policy != "never" && policy != "interval" {
		return nil, fmt.Errorf("不支持的 sub-fetch-policy: %s（可选 always、never、interval）", policy)
	}

	cachePath, err := subscriptionCachePath(urlKey)
	if err != nil {
		return nil, err
	}
	lockValue, _ := subscriptionCacheLocks.LoadOrStore(cachePath, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	cached, cacheInfo, cacheErr := readSubscriptionCache(cachePath)
	if policy == "never" {
		if cacheErr != nil {
			return nil, fmt.Errorf("禁止拉取原始订阅，且本地缓存不存在或不可读：%w", cacheErr)
		}
		return cached, nil
	}
	if policy == "interval" && cacheErr == nil && time.Since(cacheInfo.ModTime()) < subscriptionCacheMaxAge() {
		return cached, nil
	}

	data, fetchErr := FetchSubsData(urlKey)
	if fetchErr == nil {
		if err := writeSubscriptionCache(cachePath, data); err != nil {
			// A successful subscription fetch should still be usable if the cache
			// volume is read-only or temporarily unavailable.
			return data, nil
		}
		return data, nil
	}
	if cacheErr == nil {
		return cached, nil
	}
	return nil, fetchErr
}

func subscriptionCacheMaxAge() time.Duration {
	value := strings.TrimSpace(config.GlobalConfig.SubFetchInterval)
	if value == "" {
		return 12 * time.Hour
	}
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return 12 * time.Hour
	}
	return d
}

func subscriptionCachePath(url string) (string, error) {
	base := strings.TrimSpace(config.GlobalConfig.ConfigDir)
	if base == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("无法定位订阅缓存目录: %w", err)
		}
		base = filepath.Join(cacheDir, "subs-check-pro")
	}
	dir := filepath.Join(base, "subscription-cache")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("创建订阅缓存目录失败: %w", err)
	}
	hash := sha256.Sum256([]byte(url))
	return filepath.Join(dir, hex.EncodeToString(hash[:])+".cache"), nil
}

func readSubscriptionCache(path string) ([]byte, os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	if len(data) == 0 {
		return nil, nil, errors.New("订阅缓存为空")
	}
	return data, info, nil
}

func writeSubscriptionCache(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".subscription-*.tmp")
	if err != nil {
		return err
	}
	tempPath := f.Name()
	defer os.Remove(tempPath)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
