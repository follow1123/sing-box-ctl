package webui

import (
	"encoding/json"
	"fmt"
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

// postProviderMultipart 以 multipart 表单一步创建 provider（与真实前端一致）
func postProviderMultipart(t *testing.T, ts *httptest.Server, name, url, source, content string) (int, string) {
	t.Helper()
	body := &strings.Builder{}
	w := multipart.NewWriter(body)
	require.NoError(t, w.WriteField("name", name))
	require.NoError(t, w.WriteField("source", source))
	if url != "" {
		require.NoError(t, w.WriteField("url", url))
	}
	if content != "" {
		fw, err := w.CreateFormFile("file", "sub.yaml")
		require.NoError(t, err)
		_, err = fw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/providers", strings.NewReader(body.String()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(data)
}

// clashYAML 生成最小可用订阅（节点名中性占位）
func clashYAML(tag string) string {
	return fmt.Sprintf("proxies:\n  - name: %s\n    type: ss\n    server: 1.2.3.4\n    port: 8388\n    cipher: aes-128-gcm\n    password: x\n", tag)
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

	// multipart 一步创建 upload 来源 provider（首版内容随创建一起带上）
	code, body := postProviderMultipart(t, ts, "p", "", "upload", clashYAML("节点-0"))
	require.Equal(t, http.StatusCreated, code)
	var created struct {
		Uuid string `json:"uuid"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &created))
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

	// 再上传两版产生 last/old
	upload(clashYAML("节点-1"))
	upload(clashYAML("节点-2"))

	code, vBody := getBody(t, ts.URL+"/api/providers/"+uuid+"/versions")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, `["current","last","old"]`, strings.TrimSpace(vBody))

	// 还原 last
	r, err := http.Post(ts.URL+"/api/providers/"+uuid+"/restore?version=old", "", nil)
	require.NoError(t, err)
	r.Body.Close()
	require.Equal(t, http.StatusOK, r.StatusCode)
}

// 上传内容格式不对时不应创建任何 provider（不留空壳条目）
func TestProviderCreateRejectsInvalidContent(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	cases := []struct {
		name, url, source, content, wantErr string
	}{
		{"bad-yaml", "", "upload", "not a valid yaml", "unmarshal clash yaml"},
		{"empty-proxies", "", "upload", "proxies: []\n", "no valid proxies"},
		{"upload-no-file", "", "upload", "", "get file from form"},
		{"url-no-url", "", "url", "", "url is required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, body := postProviderMultipart(t, ts, c.name, c.url, c.source, c.content)
			require.Equal(t, http.StatusBadRequest, code)
			require.Contains(t, body, c.wantErr)
		})
	}

	// 列表仍为空
	code, listBody := getBody(t, ts.URL+"/api/providers")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "[]", strings.TrimSpace(listBody))
}

// url 来源一步创建：后端直接下载订阅并入库节点
func TestProviderCreateWithURL(t *testing.T) {
	sub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, clashYAML("节点-x"))
	}))
	defer sub.Close()

	ts := newTestServer(t)
	defer ts.Close()

	code, body := postProviderMultipart(t, ts, "url-p", sub.URL+"/sub.yaml", "url", "")
	require.Equal(t, http.StatusCreated, code)
	var created struct {
		Uuid   string `json:"uuid"`
		Source string `json:"source"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &created))
	require.Equal(t, "url", created.Source)

	// 一个版本已入库
	vcode, vBody := getBody(t, ts.URL+"/api/providers/"+created.Uuid+"/versions")
	require.Equal(t, http.StatusOK, vcode)
	require.Equal(t, `["current"]`, strings.TrimSpace(vBody))

	// 下载失败也不创建
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer dead.Close()
	code, _ = postProviderMultipart(t, ts, "bad-url", dead.URL+"/sub.yaml", "url", "")
	require.Equal(t, http.StatusBadGateway, code)

	code, listBody := getBody(t, ts.URL+"/api/providers")
	require.Equal(t, http.StatusOK, code)
	var list []map[string]any
	require.NoError(t, json.Unmarshal([]byte(listBody), &list))
	require.Len(t, list, 1)
	require.Equal(t, "url-p", list[0]["name"])
}

// 设为默认 provider：列表带 default 标记，可切换
func TestProviderDefaultAPI(t *testing.T) {
	sub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, clashYAML("节点-x"))
	}))
	defer sub.Close()

	ts := newTestServer(t)
	defer ts.Close()

	_, body1 := postProviderMultipart(t, ts, "p1", sub.URL, "url", "")
	_, body2 := postProviderMultipart(t, ts, "p2", sub.URL, "url", "")
	var p1, p2 struct {
		Uuid string `json:"uuid"`
	}
	require.NoError(t, json.Unmarshal([]byte(body1), &p1))
	require.NoError(t, json.Unmarshal([]byte(body2), &p2))

	listDefault := func() map[string]bool {
		code, lb := getBody(t, ts.URL+"/api/providers")
		require.Equal(t, http.StatusOK, code)
		var list []map[string]any
		require.NoError(t, json.Unmarshal([]byte(lb), &list))
		res := map[string]bool{}
		for _, it := range list {
			res[it["uuid"].(string)] = it["default"].(bool)
		}
		return res
	}

	// 初始都非默认
	flags := listDefault()
	require.False(t, flags[p1.Uuid])
	require.False(t, flags[p2.Uuid])

	// 设 p1 默认
	resp, err := http.Post(ts.URL+"/api/providers/"+p1.Uuid+"/default", "", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	flags = listDefault()
	require.True(t, flags[p1.Uuid])
	require.False(t, flags[p2.Uuid])

	// 切到 p2
	resp, err = http.Post(ts.URL+"/api/providers/"+p2.Uuid+"/default", "", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	flags = listDefault()
	require.False(t, flags[p1.Uuid])
	require.True(t, flags[p2.Uuid])

	// 不存在的 uuid
	resp, err = http.Post(ts.URL+"/api/providers/no-such/default", "", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
