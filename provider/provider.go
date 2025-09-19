package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/follow1123/sing-box-ctl/jsonhandler"
)

type Provider struct {
	path string
	jh   *jsonhandler.JsonHandler
}

func New(path string) (*Provider, error) {
	var jh *jsonhandler.JsonHandler
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			jh, err = jsonhandler.FromData([]byte("{}"))
			if err != nil {
				return nil, err
			}

		} else {
			return nil, fmt.Errorf("check provider config '%s' error:\n\t%w", path, err)
		}
	}
	if jh == nil {
		jh, err = jsonhandler.FromFile(path)
		if err != nil {
			return nil, err
		}
	}
	return &Provider{
		path: path,
		jh:   jh,
	}, nil
}

func (p *Provider) Add(name string, url string) error {
	providers, err := p.List()
	var setDefault bool
	if err != nil {
		setDefault = true
	}
	if len(providers) == 0 {
		setDefault = true
	}
	for _, p := range providers {
		if p.Name == name {
			return fmt.Errorf("duplicate provider name '%s'", name)
		}
	}
	if setDefault {
		if err := p.SetDefault(name); err != nil {
			return err
		}
	}
	if err := p.jh.Set("providers.-1", map[string]string{"name": name, "url": url}); err != nil {
		return err
	}
	return nil
}

func (p *Provider) Update(name string, url string) error {
	if _, err := p.Get(name); err != nil {
		return err
	}

	if err := p.jh.Set(fmt.Sprintf(`providers.#(name=="%s").url`, name), url); err != nil {
		return err
	}
	return nil
}

func (p *Provider) Delete(name string) error {
	providers, err := p.List()
	if err != nil {
		return err
	}
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
	defaultName, err := p.getDefaultName()
	if err != nil {
		return err
	}
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
	return p.deleteByIndex(idx)
}

func (p *Provider) SetDefault(name string) error {
	return p.jh.Set("default_provider", name)
}

func (p *Provider) Get(name string) (*Data, error) {
	providers, err := p.List()
	if err != nil {
		return nil, err
	}
	for _, p := range providers {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("provider '%s' not exists", name)
}

func (p *Provider) GetDefault() (*Data, error) {
	name, err := p.getDefaultName()
	if err != nil {
		return nil, err
	}
	return p.Get(name)
}

func (p *Provider) List() ([]Data, error) {
	result, exists := p.jh.GetResult("providers")
	var list []Data
	if !exists {
		return list, errors.New("no providers")
	}
	if !result.IsArray() {
		return nil, errors.New("providers must be array")
	}
	if err := json.Unmarshal([]byte(result.Raw), &list); err != nil {
		return nil, fmt.Errorf("unmarshal providers error:\n\t%w", err)
	}
	return list, nil
}

func (p *Provider) Save() error {
	if err := p.jh.Format(); err != nil {
		return err
	}
	return p.jh.SaveTo(p.path)
}

func (p *Provider) deleteByIndex(idx int) error {
	return p.jh.Delete(fmt.Sprintf("providers.%d", idx))
}

func (p *Provider) getDefaultName() (string, error) {
	defaultName, exists := p.jh.GetString("default_provider")
	if !exists {
		return "", errors.New("no default provider")
	}
	return defaultName, nil
}

func (p *Provider) deleteDefaultProvider() error {
	return p.jh.Delete("default_provider")
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
