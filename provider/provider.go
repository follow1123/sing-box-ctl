package provider

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// readFileContent 读取文件内容；文件不存在时返回 false
func readFileContent(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// DataFromSource 从 url 或本地文件读取数据
func DataFromSource(source string) ([]byte, error) {
	var data bytes.Buffer
	if isHTTPURL(source) {
		resp, err := http.Get(source)
		if err != nil {
			return nil, fmt.Errorf("failed to download from URL:\n\t%w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad response from server: %s", resp.Status)
		}

		_, err = io.Copy(&data, resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body:\n\t%w", err)
		}
	} else {
		absPath, err := filepath.Abs(source)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve file path:\n\t%w", err)
		}
		d, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file:\n\t%w", err)
		}
		_, err = data.Write(d)
		if err != nil {
			return nil, fmt.Errorf("save data to buffer error:\n\t%w", err)
		}
	}
	return data.Bytes(), nil
}

// 判断是否是 HTTP/HTTPS URL
func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
