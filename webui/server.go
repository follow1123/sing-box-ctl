package webui

import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/provider"
	"github.com/follow1123/sing-box-ctl/settings"
)

//go:embed dist
var distFiles embed.FS

//go:embed default_template.json
var templateSeedData []byte

const (
	apiPath          = "/api/providers"
	apiTemplatesPath = "/api/templates"
	configPathURL    = "/config/"
	// BuiltinTemplateKey 内嵌种子模板的特殊 key（不落盘，仅作为新建模板的来源）
	BuiltinTemplateKey = "builtin"
)

type Server struct {
	workingDir string
	certFile   string
	keyFile    string
	pm         *provider.ProviderManager
	server     *http.Server
}

// Options 服务启动配置（由 cmd 层从命令行与 working_dir/config.json 合并而来）
type Options struct {
	WorkingDir string
	Listen     string
	Port       int
	CertFile   string
	KeyFile    string
}

func New(opts Options) (*Server, error) {
	// 先解析为绝对路径，便于后续日志与每次请求重建 provider 时保持一致
	p, err := provider.New(opts.WorkingDir)
	if err != nil {
		return nil, err
	}
	absDir := p.WorkingDir()
	if err := initWorkingDir(absDir); err != nil {
		return nil, err
	}
	pm, err := provider.NewManager(absDir)
	if err != nil {
		return nil, err
	}
	s := &Server{workingDir: absDir, certFile: opts.CertFile, keyFile: opts.KeyFile, pm: pm}

	mux := http.NewServeMux()
	mux.HandleFunc(apiPath, s.providersHandle)
	mux.HandleFunc(apiPath+"/", s.providerHandle)
	mux.HandleFunc(apiTemplatesPath, s.templatesHandle)
	mux.HandleFunc(apiTemplatesPath+"/", s.templateHandle)
	mux.HandleFunc(configPathURL, s.configHandle)
	// 前端构建产物（Vite 输出）
	assetsSub, err := fs.Sub(distFiles, "dist/assets")
	if err != nil {
		return nil, fmt.Errorf("init dist files error:\n\t%w", err)
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsSub))))
	// SPA：其余路径返回 index.html
	mux.HandleFunc("/", s.spaHandle)

	s.server = &http.Server{
		Addr:    net.JoinHostPort(opts.Listen, strconv.Itoa(opts.Port)),
		Handler: recoverMiddleware(mux),
	}
	return s, nil
}

// initWorkingDir 初始化工作目录（providers/templates 目录），并保证存在一个默认模板
func initWorkingDir(workingDir string) error {
	p, err := provider.New(workingDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p.ProvidersDir(), 0700); err != nil {
		return fmt.Errorf("init providers dir error:\n\t%w", err)
	}
	if err := os.MkdirAll(p.TemplatesDir(), 0700); err != nil {
		return fmt.Errorf("init templates dir error:\n\t%w", err)
	}
	return nil
}

func (s *Server) Serve() error {
	log.Printf("working directory: %s", s.workingDir)
	// 先绑定端口，成功后再打印启动日志，避免 bind 失败时日志顺序误导
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("listen %s error:\n\t%w", s.server.Addr, err)
	}
	scheme := "http"
	if s.certFile != "" {
		// 配置了证书则启用 https（证书文件已在启动前校验过）
		scheme = "https"
		log.Printf("tls certificate: %s", s.certFile)
	}
	log.Printf("webui started on %s://%s", scheme, s.server.Addr)
	var serveErr error
	if s.certFile != "" {
		serveErr = s.server.ServeTLS(ln, s.certFile, s.keyFile)
	} else {
		serveErr = s.server.Serve(ln)
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return fmt.Errorf("start webui server error:\n\t%w", serveErr)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// recoverMiddleware 捕获 handler 中的 panic，防止服务崩溃
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				handleError(w, fmt.Errorf("internal error: %v", err), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) newProvider() (*provider.Provider, error) {
	return provider.New(s.workingDir)
}

// spaHandle 返回前端入口（Vite 构建的 index.html）
func (s *Server) spaHandle(w http.ResponseWriter, r *http.Request) {
	data, err := distFiles.ReadFile("dist/index.html")
	if err != nil {
		handleError(w, fmt.Errorf("frontend not built, run: cd frontend && pnpm build"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// ================= provider API =================

type providerRequest struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Source  string `json:"source"`
	Message string `json:"message"`
}

func decodeProviderRequest(r *http.Request) (*providerRequest, error) {
	req := &providerRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return nil, fmt.Errorf("decode request body error:\n\t%w", err)
	}
	return req, nil
}

// GET /api/providers -> 列表
// POST /api/providers -> multipart 一步创建：上传文件或提供 url，解析出节点成功后才创建
//
//	form: name/source/message/url；source=upload 时必须带 file，source=url 时必须带 url
func (s *Server) providersHandle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.pm.List())
	case http.MethodPost:
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB 上限
			handleError(w, fmt.Errorf("parse multipart form error:\n\t%w", err), http.StatusBadRequest)
			return
		}
		name := r.FormValue("name")
		source := r.FormValue("source")
		message := r.FormValue("message")
		url := r.FormValue("url")
		if source == "" {
			source = provider.SourceURL
		}

		// 先取内容（上传文件或下载 url）并解析出节点，失败则不落盘
		var raw []byte
		var fileName string
		switch source {
		case provider.SourceUpload:
			f, header, err := r.FormFile("file")
			if err != nil {
				handleError(w, fmt.Errorf("get file from form error:\n\t%w", err), http.StatusBadRequest)
				return
			}
			defer f.Close()
			raw, err = io.ReadAll(f)
			if err != nil {
				handleError(w, fmt.Errorf("read uploaded file error:\n\t%w", err), http.StatusBadRequest)
				return
			}
			fileName = header.Filename
		case provider.SourceURL:
			if url == "" {
				handleError(w, fmt.Errorf("url is required when source is '%s'", provider.SourceURL), http.StatusBadRequest)
				return
			}
			data, err := provider.DataFromSource(url)
			if err != nil {
				handleError(w, err, http.StatusBadGateway)
				return
			}
			raw = data
		default:
			handleError(w, fmt.Errorf("invalid source '%s', must be '%s' or '%s'", source, provider.SourceURL, provider.SourceUpload), http.StatusBadRequest)
			return
		}
		nodes, err := converter.NodesFromClash(raw)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		nodeJSON, err := json.Marshal(nodes)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}

		// 解析成功后才创建 provider，避免留下无内容的空壳条目
		uuid, err := s.pm.Add(name, url, source, message)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		if err := s.pm.SaveNodes(uuid, nodeJSON); err != nil {
			s.pm.Delete(uuid)
			handleInternalServerError(w, err)
			return
		}
		if source == provider.SourceUpload {
			if err := s.pm.SetFileName(uuid, fileName); err != nil {
				s.pm.Delete(uuid)
				handleInternalServerError(w, err)
				return
			}
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, s.pm.Get(uuid))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// PUT /api/providers/<uuid> -> 更新
// DELETE /api/providers/<uuid> -> 删除
// POST /api/providers/<uuid>/fetch -> 下载订阅
// POST /api/providers/<uuid>/upload -> 上传配置
func (s *Server) providerHandle(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, apiPath+"/")
	parts := strings.Split(rest, "/")
	if len(parts) < 1 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	uuid := parts[0]

	switch {
	case len(parts) == 1 && r.Method == http.MethodPut:
		req, err := decodeProviderRequest(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		if err := s.pm.Update(uuid, req.Name, req.Url, req.Source, req.Message); err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		writeJSON(w, s.pm.Get(uuid))
	case len(parts) == 1 && r.Method == http.MethodDelete:
		if err := s.pm.Delete(uuid); err != nil {
			handleInternalServerError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 2 && parts[1] == "fetch" && r.Method == http.MethodPost:
		s.fetchHandle(w, r, uuid)
	case len(parts) == 2 && parts[1] == "upload" && r.Method == http.MethodPost:
		s.uploadHandle(w, r, uuid)
	case len(parts) == 2 && parts[1] == "versions" && r.Method == http.MethodGet:
		if s.pm.Get(uuid) == nil {
			handleError(w, fmt.Errorf("no provider with uuid: %s", uuid), http.StatusNotFound)
			return
		}
		vers, err := s.pm.SubscriptionVersions(uuid)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		writeJSON(w, vers)
	case len(parts) == 2 && parts[1] == "restore" && r.Method == http.MethodPost:
		if s.pm.Get(uuid) == nil {
			handleError(w, fmt.Errorf("no provider with uuid: %s", uuid), http.StatusNotFound)
			return
		}
		if err := s.pm.RestoreSubscription(uuid, r.URL.Query().Get("version")); err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		log.Printf("provider '%s' restored to version: %s", uuid, r.URL.Query().Get("version"))
		writeJSON(w, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// storeSubscriptionNodes 下载/上传的原文经抠节点转换为 sing-box outbounds 后滚动入库
func (s *Server) storeSubscriptionNodes(w http.ResponseWriter, uuid string, data []byte) error {
	nodes, err := converter.NodesFromClash(data)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return err
	}
	nodeJSON, err := json.Marshal(nodes)
	if err != nil {
		handleInternalServerError(w, err)
		return err
	}
	if err := s.pm.SaveNodes(uuid, nodeJSON); err != nil {
		handleInternalServerError(w, err)
		return err
	}
	return nil
}

// 下载订阅并保存（下载 -> 抠出节点转 sing-box outbounds -> 滚动入库）
func (s *Server) fetchHandle(w http.ResponseWriter, r *http.Request, uuid string) {
	prov := s.pm.Get(uuid)
	if prov == nil {
		handleError(w, fmt.Errorf("no provider with uuid: %s", uuid), http.StatusNotFound)
		return
	}
	if prov.Source != provider.SourceURL || prov.Url == "" {
		handleError(w, fmt.Errorf("provider '%s' has no url, use upload instead", prov.Name), http.StatusBadRequest)
		return
	}

	data, err := provider.DataFromSource(prov.Url)
	if err != nil {
		handleError(w, err, http.StatusBadGateway)
		return
	}
	if err := s.storeSubscriptionNodes(w, uuid, data); err != nil {
		return
	}
	log.Printf("provider '%s' subscription updated: %d bytes", prov.Name, len(data))
	writeJSON(w, map[string]any{"ok": true, "bytes": len(data)})
}

// 上传配置文件并保存（原文 -> 抠出节点转 sing-box outbounds -> 滚动入库）
func (s *Server) uploadHandle(w http.ResponseWriter, r *http.Request, uuid string) {
	prov := s.pm.Get(uuid)
	if prov == nil {
		handleError(w, fmt.Errorf("no provider with uuid: %s", uuid), http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB 上限
		handleError(w, fmt.Errorf("parse multipart form error:\n\t%w", err), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		handleError(w, fmt.Errorf("get file from form error:\n\t%w", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		handleError(w, fmt.Errorf("read uploaded file error:\n\t%w", err), http.StatusBadRequest)
		return
	}
	if len(data) == 0 {
		handleError(w, fmt.Errorf("uploaded file is empty"), http.StatusBadRequest)
		return
	}
	if err := s.storeSubscriptionNodes(w, uuid, data); err != nil {
		return
	}
	// 记录上传文件名
	if err := s.pm.SetFileName(uuid, header.Filename); err != nil {
		handleInternalServerError(w, err)
		return
	}
	log.Printf("provider '%s' subscription uploaded: %d bytes", prov.Name, len(data))
	writeJSON(w, map[string]any{"ok": true, "bytes": len(data), "file_name": s.pm.Get(uuid).FileName})
}

// ================= template API =================

// isValidTemplateUuid 校验模板 uuid：非空、无路径分隔符
func isValidTemplateUuid(uuid string) bool {
	return uuid != "" && filepath.Base(uuid) == uuid && !strings.HasPrefix(uuid, ".")
}

// GET /api/templates -> 模板列表 [{uuid,name,default}]
// POST /api/templates?name=xxx&from=builtin|<uuid> -> 新建模板（复制来源内容）
func (s *Server) templatesHandle(w http.ResponseWriter, r *http.Request) {
	p, err := s.newProvider()
	if err != nil {
		handleInternalServerError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		infos, err := p.ListTemplates()
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		writeJSON(w, infos)
	case http.MethodPost:
		name := r.URL.Query().Get("name")
		if name == "" {
			handleError(w, fmt.Errorf("name is required"), http.StatusBadRequest)
			return
		}
		source, err := resolveTemplateSource(p, r.URL.Query().Get("from"))
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		uuid, err := p.AddTemplate(name)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		// 内容来自来源（内置种子或已有用户模板）
		if err := p.SaveTemplate(uuid, source); err != nil {
			handleInternalServerError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, p.GetTemplate(uuid))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// resolveTemplateSource 解析新建模板的来源内容：缺省或 from=builtin 使用内嵌种子模板；
// from=<uuid> 复制已有用户模板内容
func resolveTemplateSource(p *provider.Provider, from string) ([]byte, error) {
	if from == "" || from == BuiltinTemplateKey {
		return templateSeedData, nil
	}
	if !isValidTemplateUuid(from) || !p.TemplateExists(from) {
		return nil, fmt.Errorf("template source not found: %s", from)
	}
	data, err := p.ReadTemplate(from)
	if err != nil {
		return nil, fmt.Errorf("read template source error:\n\t%w", err)
	}
	return data, nil
}

// GET /api/templates/<uuid> -> 模板内容
// PUT /api/templates/<uuid> -> 保存模板（body 为 JSON 内容，滚动 current/last/old）
// DELETE /api/templates/<uuid> -> 删除模板（默认模板不可删）
// POST /api/templates/<uuid>/default -> 设为默认
func (s *Server) templateHandle(w http.ResponseWriter, r *http.Request) {
	p, err := s.newProvider()
	if err != nil {
		handleInternalServerError(w, err)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, apiTemplatesPath+"/")
	parts := strings.Split(rest, "/")
	if len(parts) < 1 || !isValidTemplateUuid(parts[0]) {
		handleError(w, fmt.Errorf("invalid template uuid"), http.StatusBadRequest)
		return
	}
	uuid := parts[0]
	info := p.GetTemplate(uuid)
	if info == nil {
		handleError(w, fmt.Errorf("no template with uuid: %s", uuid), http.StatusNotFound)
		return
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		data, err := p.ReadTemplate(uuid)
		if err != nil {
			handleError(w, err, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(data)
	case len(parts) == 1 && r.Method == http.MethodPut:
		data, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, fmt.Errorf("read request body error:\n\t%w", err), http.StatusBadRequest)
			return
		}
		if !json.Valid(data) {
			handleError(w, fmt.Errorf("invalid json content"), http.StatusBadRequest)
			return
		}
		if err := p.SaveTemplate(uuid, data); err != nil {
			handleInternalServerError(w, err)
			return
		}
		log.Printf("template '%s' saved: %d bytes", info.Name, len(data))
		writeJSON(w, map[string]any{"ok": true})
	case len(parts) == 1 && r.Method == http.MethodDelete:
		if info.Default {
			handleError(w, fmt.Errorf("default template cannot be deleted"), http.StatusBadRequest)
			return
		}
		if err := p.DeleteTemplate(uuid); err != nil {
			handleInternalServerError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 2 && parts[1] == "default" && r.Method == http.MethodPost:
		if err := p.SetDefaultTemplate(uuid); err != nil {
			handleInternalServerError(w, err)
			return
		}
		writeJSON(w, p.GetTemplate(uuid))
	case len(parts) == 2 && parts[1] == "versions" && r.Method == http.MethodGet:
		vers, err := p.TemplateVersions(uuid)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		writeJSON(w, vers)
	case len(parts) == 2 && parts[1] == "restore" && r.Method == http.MethodPost:
		if err := p.RestoreTemplate(uuid, r.URL.Query().Get("version")); err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		log.Printf("template '%s' restored to version: %s", uuid, r.URL.Query().Get("version"))
		writeJSON(w, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ================= config API =================

// GET /config/<uuid>?template=<template-uuid>&platform=&tun&mixed&... -> 实时转换订阅为 sing-box 配置
func (s *Server) configHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	uuid := strings.TrimPrefix(r.URL.Path, configPathURL)
	if uuid == "" {
		http.NotFound(w, r)
		return
	}

	p, err := s.newProvider()
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	if s.pm.Get(uuid) == nil {
		handleError(w, fmt.Errorf("no provider with uuid: %s", uuid), http.StatusNotFound)
		return
	}
	// 读取当前节点（sing-box outbounds）
	nodesData, err := s.pm.ReadSubscription(uuid)
	if err != nil {
		handleError(w, err, http.StatusNotFound)
		return
	}
	var nodes []map[string]any
	if err := json.Unmarshal(nodesData, &nodes); err != nil {
		handleError(w, fmt.Errorf("parse provider nodes error: %w", err), http.StatusBadRequest)
		return
	}

	// 选择模板（uuid，缺省用默认模板）
	template := r.URL.Query().Get("template")
	if template == "" {
		template, err = p.DefaultTemplate()
		if err != nil {
			handleError(w, fmt.Errorf("no template available, create one first"), http.StatusNotFound)
			return
		}
	}
	if !isValidTemplateUuid(template) || !p.TemplateExists(template) {
		handleError(w, fmt.Errorf("template not found"), http.StatusNotFound)
		return
	}

	// 读取模板原始配置（settings 需要默认 mixed/tun inbound、api service）
	tmplData, err := p.ReadTemplate(template)
	if err != nil {
		handleError(w, err, http.StatusNotFound)
		return
	}
	tmplConf := &converter.SingBox{}
	if err := json.Unmarshal(tmplData, tmplConf); err != nil {
		handleError(w, fmt.Errorf("parse template error: %w", err), http.StatusBadRequest)
		return
	}

	// 实时转换订阅为 sing-box 配置
	conv, err := converter.New(filepath.Join(p.TemplateDir(template), "current"))
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	sb, err := conv.Build(nodes)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}

	platform := r.URL.Query().Get("platform")

	// 解析 URL 参数并应用到转换结果（内存中，无中间文件）
	values, err := parseConfigQuery(r.URL.Query())
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}
	// Android 必须使用 tun：即使 URL 中删除了 tun 参数也强制启用
	if platform == "android" {
		values[settings.StTunStatus] = "true"
	}
	st := settings.New(tmplConf, sb)
	if err := st.SetMap(values); err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}
	if platform != "" {
		if err := st.SetPlatform(platform); err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
	}

	// 校验：输出配置至少需要一个 inbound（tun/mixed 至少启用其一，模板常驻 inbound 除外）
	if len(st.GetConfig().Inbounds) == 0 {
		handleError(w, fmt.Errorf("no inbound in output config, enable at least one of tun/mixed"), http.StatusBadRequest)
		return
	}

	jsonData, err := st.ToJson(true)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(jsonData)
}

// parseConfigQuery 将 URL 查询参数映射为 settings 设置项
func parseConfigQuery(q url.Values) (map[settings.SettingName]string, error) {
	values := make(map[settings.SettingName]string)
	if _, ok := q["tun"]; ok {
		values[settings.StTunStatus] = "true"
	}
	if _, ok := q["mixed"]; ok {
		values[settings.StMixedStatus] = "true"
	}
	if v := q.Get("mixed-listen"); v != "" {
		values[settings.StMixedListen] = v
	}
	if v := q.Get("mixed-port"); v != "" {
		values[settings.StMixedPort] = v
	}
	if _, ok := q["sys-proxy"]; ok {
		values[settings.StMixedSysProxyStatus] = "true"
	}
	// sing-box api service（services.api），dashboard 为其子配置
	if _, ok := q["api"]; ok {
		values[settings.StAPIStatus] = "true"
	}
	if v := q.Get("api-listen"); v != "" {
		values[settings.StAPIListen] = v
	}
	if v := q.Get("api-port"); v != "" {
		values[settings.StAPIPort] = v
	}
	if v := q.Get("api-secret"); v != "" {
		values[settings.StAPISecret] = v
	}
	if v := q.Get("api-dashboard"); v != "" {
		values[settings.StAPIDashboard] = v
	}
	return values, nil
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Print(err)
	}
}

func handleError(w http.ResponseWriter, err error, code int) {
	log.Print(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
}

func handleInternalServerError(w http.ResponseWriter, err error) {
	handleError(w, fmt.Errorf("Internal Server Error\n\t%w", err), http.StatusInternalServerError)
}
