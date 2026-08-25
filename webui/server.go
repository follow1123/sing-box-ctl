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
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/provider"
)

//go:embed index.html
var indexHtml string

//go:embed template.html
var templateHtml string

//go:embed static/*
var staticFiles embed.FS

const (
	homeUrlPath     = "/"
	templatesUrlPath = "/templates"
	apiPath         = "/api/providers"
	apiTemplatesPath = "/api/templates"
	configPathURL   = "/config/"
	staticUrlPath   = "/static/"
)

type Server struct {
	configPath string
	conf       *config.Config
	server     *http.Server
}

func New(configPath string, port int) (*Server, error) {
	conf, err := config.New(configPath)
	if err != nil {
		return nil, err
	}
	s := &Server{configPath: configPath, conf: conf}

	mux := http.NewServeMux()
	mux.HandleFunc(homeUrlPath, s.homeHandle)
	mux.HandleFunc(templatesUrlPath, s.templatesPageHandle)
	mux.HandleFunc(apiPath, s.providersHandle)
	mux.HandleFunc(apiPath+"/", s.providerHandle)
	mux.HandleFunc(apiTemplatesPath, s.templatesHandle)
	mux.HandleFunc(apiTemplatesPath+"/", s.templateHandle)
	mux.HandleFunc(configPathURL, s.configHandle)
	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("init static files error:\n\t%w", err)
	}
	mux.Handle(staticUrlPath, http.StripPrefix(staticUrlPath, http.FileServer(http.FS(staticSub))))

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: recoverMiddleware(mux),
	}
	return s, nil
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

func (s *Server) Serve() error {
	log.Printf("webui started on %s", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("start webui server error:\n\t%w", err)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) newProvider() (*provider.Provider, error) {
	return provider.New(s.configPath)
}

func (s *Server) homeHandle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != homeUrlPath {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHtml))
}

func (s *Server) templatesPageHandle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != templatesUrlPath {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(templateHtml))
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
// POST /api/providers -> 添加
func (s *Server) providersHandle(w http.ResponseWriter, r *http.Request) {
	p, err := s.newProvider()
	if err != nil {
		handleInternalServerError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, p.List())
	case http.MethodPost:
		req, err := decodeProviderRequest(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		if err := p.Add(req.Name, req.Url, req.Source, req.Message); err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		if err := p.Save(); err != nil {
			handleInternalServerError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, p.List())
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// PUT /api/providers/<uuid> -> 更新
// DELETE /api/providers/<uuid> -> 删除
// POST /api/providers/<uuid>/fetch -> 下载订阅
// POST /api/providers/<uuid>/upload -> 上传配置
func (s *Server) providerHandle(w http.ResponseWriter, r *http.Request) {
	p, err := s.newProvider()
	if err != nil {
		handleInternalServerError(w, err)
		return
	}

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
		if err := p.Update(uuid, req.Name, req.Url, req.Source, req.Message); err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}
		if err := p.Save(); err != nil {
			handleInternalServerError(w, err)
			return
		}
		writeJSON(w, p.Get(uuid))
	case len(parts) == 1 && r.Method == http.MethodDelete:
		if err := p.Delete(uuid); err != nil {
			handleInternalServerError(w, err)
			return
		}
		if err := p.Save(); err != nil {
			handleInternalServerError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 2 && parts[1] == "fetch" && r.Method == http.MethodPost:
		s.fetchHandle(w, r, p, uuid)
	case len(parts) == 2 && parts[1] == "upload" && r.Method == http.MethodPost:
		s.uploadHandle(w, r, p, uuid)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// 下载订阅并保存（滚动 current/last/old）
func (s *Server) fetchHandle(w http.ResponseWriter, r *http.Request, p *provider.Provider, uuid string) {
	prov := p.Get(uuid)
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
	if err := p.SaveSubscription(uuid, data); err != nil {
		handleInternalServerError(w, err)
		return
	}
	log.Printf("provider '%s' subscription updated: %d bytes", prov.Name, len(data))
	writeJSON(w, map[string]any{"ok": true, "bytes": len(data)})
}

// 上传配置文件并保存（滚动 current/last/old）
func (s *Server) uploadHandle(w http.ResponseWriter, r *http.Request, p *provider.Provider, uuid string) {
	prov := p.Get(uuid)
	if prov == nil {
		handleError(w, fmt.Errorf("no provider with uuid: %s", uuid), http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB 上限
		handleError(w, fmt.Errorf("parse multipart form error:\n\t%w", err), http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
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
	if err := p.SaveSubscription(uuid, data); err != nil {
		handleInternalServerError(w, err)
		return
	}
	log.Printf("provider '%s' subscription uploaded: %d bytes", prov.Name, len(data))
	writeJSON(w, map[string]any{"ok": true, "bytes": len(data)})
}

// ================= template API =================

// isValidTemplateName 校验模板名：非空、无路径分隔符、不以 . 开头、不带 .json 后缀
func isValidTemplateName(name string) bool {
	return name != "" &&
		name != "." && name != ".." &&
		!strings.ContainsAny(name, `/\`) &&
		!strings.HasPrefix(name, ".") &&
		!strings.HasSuffix(name, ".json")
}

func (s *Server) templatePath(name string) string {
	return filepath.Join(s.conf.TemplatesDir, name+".json")
}

// GET /api/templates -> 模板名列表
// POST /api/templates?name=xxx -> 新建模板（初始化为默认内容）
func (s *Server) templatesHandle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		entries, err := os.ReadDir(s.conf.TemplatesDir)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		var names []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				names = append(names, strings.TrimSuffix(e.Name(), ".json"))
			}
		}
		sort.Strings(names)
		writeJSON(w, names)
	case http.MethodPost:
		name := r.URL.Query().Get("name")
		if !isValidTemplateName(name) {
			handleError(w, fmt.Errorf("invalid template name: %s", name), http.StatusBadRequest)
			return
		}
		path := s.templatePath(name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			handleError(w, fmt.Errorf("template '%s' already exists", name), http.StatusConflict)
			return
		}
		// 复制默认模板内容作为初始内容
		if err := os.WriteFile(path, defaultTemplateData(), 0600); err != nil {
			handleInternalServerError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]any{"name": name})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GET /api/templates/<name> -> 模板内容
// PUT /api/templates/<name> -> 保存模板（body 为 JSON 内容）
// DELETE /api/templates/<name> -> 删除模板
func (s *Server) templateHandle(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, apiTemplatesPath+"/")
	if !isValidTemplateName(name) {
		handleError(w, fmt.Errorf("invalid template name: %s", name), http.StatusBadRequest)
		return
	}
	path := s.templatePath(name)

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(path)
		if err != nil {
			handleError(w, err, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(data)
	case http.MethodPut:
		data, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, fmt.Errorf("read request body error:\n\t%w", err), http.StatusBadRequest)
			return
		}
		// 校验是合法 JSON
		if !json.Valid(data) {
			handleError(w, fmt.Errorf("invalid json content"), http.StatusBadRequest)
			return
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			handleInternalServerError(w, err)
			return
		}
		log.Printf("template '%s' saved: %d bytes", name, len(data))
		writeJSON(w, map[string]any{"ok": true})
	case http.MethodDelete:
		if name == config.DefaultTemplateName {
			handleError(w, fmt.Errorf("default template cannot be deleted"), http.StatusBadRequest)
			return
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			handleInternalServerError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ================= config API =================

// GET /config/<uuid>?template=<name> -> 实时转换订阅为 sing-box 配置
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
	if p.Get(uuid) == nil {
		handleError(w, fmt.Errorf("no provider with uuid: %s", uuid), http.StatusNotFound)
		return
	}
	// 读取当前订阅
	data, err := p.ReadSubscription(uuid)
	if err != nil {
		handleError(w, err, http.StatusNotFound)
		return
	}

	// 选择模板
	template := r.URL.Query().Get("template")
	if template == "" {
		template = config.DefaultTemplateName
	}
	if !isValidTemplateName(template) {
		handleError(w, fmt.Errorf("invalid template name: %s", template), http.StatusBadRequest)
		return
	}
	templatePath := s.templatePath(template)
	if _, err := os.Stat(templatePath); err != nil {
		handleError(w, fmt.Errorf("template '%s' not found", template), http.StatusNotFound)
		return
	}

	// 实时转换
	conv, err := converter.New(templatePath)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	sb, err := conv.Convert(data)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}
	jsonData, err := json.MarshalIndent(sb, "", "  ")
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(jsonData)
}

// defaultTemplateData 返回默认模板内容（从 config 包 embed 数据）
func defaultTemplateData() []byte {
	return config.TemplateData()
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
