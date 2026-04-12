package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
)

type Provider struct {
	path   string
	config *SingBoxCtlConfig
}

func New(path string) (*Provider, error) {
	config := &SingBoxCtlConfig{}
	data, err := os.ReadFile(path)
	var notExists bool
	if err != nil {
		if os.IsNotExist(err) {
			notExists = true
		} else {
			return nil, fmt.Errorf("check provider config '%s' error:\n\t%w", path, err)
		}
	}
	if !notExists {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("unmarshal json error:\n\t%w", err)
		}

	}
	return &Provider{
		path:   path,
		config: config,
	}, nil
}

func (p *Provider) Add(name string, url string) error {
	providers := p.config.Providers
	var setDefault bool
	if len(providers) == 0 {
		setDefault = true
	}
	for _, p := range providers {
		if p.Name == name {
			return fmt.Errorf("duplicate provider name '%s'", name)
		}
	}
	if setDefault {
		p.SetDefault(name)
	}
	p.config.Providers = append(p.config.Providers, ProviderConfig{
		Name: name,
		Url:  url,
	})
	return nil
}

func (p *Provider) Update(name string, url string) error {
	var idx int = -1
	for i, p := range p.config.Providers {
		if p.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("no provider named: %s", name)
	} else {
		p.config.Providers[idx].Url = url
	}
	return nil
}

func (p *Provider) Delete(name string) error {
	providers := p.config.Providers
	var idx = -1
	for i, p := range providers {
		if p.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	defaultName := p.config.DefaultProvider
	// 删除的是默认 provider 修改默认为上一个
	if defaultName == providers[idx].Name {
		// 只有一个，直接删除默认 provider
		if len(providers) == 1 {
			p.deleteDefaultProvider()
		} else {
			var nextDefaultIdx = idx - 1
			if nextDefaultIdx < 0 {
				nextDefaultIdx = idx + 1
			}
			p.SetDefault(providers[nextDefaultIdx].Name)
		}
	}

	p.config.Providers = slices.Delete(p.config.Providers, idx, idx+1)
	return nil
}

func (p *Provider) SetDefault(name string) {
	p.config.DefaultProvider = name
}

func (p *Provider) Get(name string) *ProviderConfig {
	for _, p := range p.config.Providers {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func (p *Provider) GetDefault() *ProviderConfig {
	return p.Get(p.config.DefaultProvider)
}

func (p *Provider) List() []ProviderConfig {
	return p.config.Providers
}

func (p *Provider) Save() error {
	data, err := json.MarshalIndent(p.config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal to json error:\n\t%w", err)
	}

	if err := os.WriteFile(p.path, data, 0660); err != nil {
		return fmt.Errorf("save config to %s error:\n\t%w", p.path, err)
	}
	return nil
}

func (p *Provider) deleteDefaultProvider() {
	p.config.DefaultProvider = ""
}

type Data struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

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
