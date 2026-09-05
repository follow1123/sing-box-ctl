package webui

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	s, err := New(Options{WorkingDir: t.TempDir(), Listen: "127.0.0.1", Port: 0})
	require.NoError(t, err)
	return httptest.NewServer(s.server.Handler)
}

func getBody(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(data)
}

func postTemplate(t *testing.T, ts *httptest.Server, name, from string) (int, string) {
	t.Helper()
	url := ts.URL + "/api/templates?name=" + name
	if from != "" {
		url += "&from=" + from
	}
	resp, err := http.Post(url, "", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(data)
}

func TestTemplatesEmptyOnFreshWorkingDir(t *testing.T) {
	// 种子模板不再自动落盘：新工作目录的模板列表应为空
	ts := newTestServer(t)
	defer ts.Close()
	code, body := getBody(t, ts.URL+"/api/templates")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "[]", strings.TrimSpace(body))
}

func TestCreateTemplateFromBuiltin(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// 缺省 from（内置种子）
	code, body := postTemplate(t, ts, "seed-copy", "")
	require.Equal(t, http.StatusCreated, code)
	var created struct {
		Uuid    string `json:"uuid"`
		Name    string `json:"name"`
		Default bool   `json:"default"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &created))
	require.Equal(t, "seed-copy", created.Name)
	require.False(t, created.Default, "新建模板不应默认标记")

	// 内容 = 内嵌种子
	code, content := getBody(t, ts.URL+"/api/templates/"+created.Uuid)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, string(templateSeedData), content)

	// 显式 from=builtin 同理
	code2, body2 := postTemplate(t, ts, "seed-copy-2", BuiltinTemplateKey)
	require.Equal(t, http.StatusCreated, code2)
	var created2 struct {
		Uuid string `json:"uuid"`
	}
	require.NoError(t, json.Unmarshal([]byte(body2), &created2))
	_, content2 := getBody(t, ts.URL+"/api/templates/"+created2.Uuid)
	require.Equal(t, string(templateSeedData), content2)
}

func TestCreateTemplateFromUserTemplate(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// 先建一个用户模板并修改其内容
	_, body := postTemplate(t, ts, "base", "")
	var base struct {
		Uuid string `json:"uuid"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &base))

	modified := `{"log": {"level": "debug"}, "inbounds": [], "outbounds": []}`
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/templates/"+base.Uuid, strings.NewReader(modified))
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// from=base 复制其内容
	code, body2 := postTemplate(t, ts, "copy-of-base", base.Uuid)
	require.Equal(t, http.StatusCreated, code)
	var copyT struct {
		Uuid string `json:"uuid"`
	}
	require.NoError(t, json.Unmarshal([]byte(body2), &copyT))

	_, content := getBody(t, ts.URL+"/api/templates/"+copyT.Uuid)
	require.Equal(t, modified, content)

	// from=不存在的模板 -> 400
	codeBad, _ := postTemplate(t, ts, "bad", "no-such-uuid-here")
	require.Equal(t, http.StatusBadRequest, codeBad)
}

func TestDefaultFlagFlow(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	_, body := postTemplate(t, ts, "t1", "")
	_, body2 := postTemplate(t, ts, "t2", "")
	var t1, t2 struct {
		Uuid string `json:"uuid"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &t1))
	require.NoError(t, json.Unmarshal([]byte(body2), &t2))

	// 设置 t1 为默认
	resp, err := http.Post(ts.URL+"/api/templates/"+t1.Uuid+"/default", "", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 列表里 t1 default=true、t2 default=false
	code, listBody := getBody(t, ts.URL+"/api/templates")
	require.Equal(t, http.StatusOK, code)
	var list []map[string]any
	require.NoError(t, json.Unmarshal([]byte(listBody), &list))
	found := map[string]bool{}
	for _, item := range list {
		u, _ := item["uuid"].(string)
		d, _ := item["default"].(bool)
		found[u] = d
	}
	require.True(t, found[t1.Uuid])
	require.False(t, found[t2.Uuid])

	// 默认模板不可删除
	delReq, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/templates/"+t1.Uuid, nil)
	require.NoError(t, err)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	require.Equal(t, http.StatusBadRequest, delResp.StatusCode)
}

func TestTemplateVersionsRestoreAPI(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	_, body := postTemplate(t, ts, "t", "")
	var created struct {
		Uuid string `json:"uuid"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &created))
	uuid := created.Uuid

	// 保存两版以产生 last/old
	put := func(content string) {
		req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/templates/"+uuid, strings.NewReader(content))
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
	put(`{"v": 1}`)
	put(`{"v": 2}`)

	// 版本列表
	code, vBody := getBody(t, ts.URL+"/api/templates/"+uuid+"/versions")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, `["current","last","old"]`, strings.TrimSpace(vBody))

	// 还原 old（内容是种子）
	resp, err := http.Post(ts.URL+"/api/templates/"+uuid+"/restore?version=old", "", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	code, content := getBody(t, ts.URL+"/api/templates/"+uuid)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, string(templateSeedData), content)

	// 非法版本
	resp, err = http.Post(ts.URL+"/api/templates/"+uuid+"/restore?version=nope", "", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProviderVersionsRestoreAPI(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// 添加 upload 来源 provider
	reqBody := `{"name":"p","source":"upload"}`
	resp, err := http.Post(ts.URL+"/api/providers", "application/json", strings.NewReader(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created struct {
		Uuid string `json:"uuid"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	uuid := created.Uuid

	upload := func(content string) {
		body := &strings.Builder{}
		w := multipart.NewWriter(body)
		fw, err := w.CreateFormFile("file", "sub.yaml")
		require.NoError(t, err)
		fw.Write([]byte(content))
		w.Close()

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/providers/"+uuid+"/upload", strings.NewReader(body.String()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		r, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer r.Body.Close()
		require.Equal(t, http.StatusOK, r.StatusCode)
	}

	upload("c0")
	upload("c1")
	upload("c2")

	code, vBody := getBody(t, ts.URL+"/api/providers/"+uuid+"/versions")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, `["current","last","old"]`, strings.TrimSpace(vBody))

	// 还原 last
	r, err := http.Post(ts.URL+"/api/providers/"+uuid+"/restore?version=old", "", nil)
	require.NoError(t, err)
	r.Body.Close()
	require.Equal(t, http.StatusOK, r.StatusCode)
}
