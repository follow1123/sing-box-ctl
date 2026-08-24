package webui

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/provider"
)

//go:embed index.html
var indexHtml string

const (
	homeUrlPath   = "/"
	apiPath       = "/api/providers"
	configPathURL = "/config/"
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
	mux.HandleFunc(apiPath, s.providersHandle)
	mux.HandleFunc(apiPath+"/", s.providerHandle)
	mux.HandleFunc(configPathURL, s.configHandle)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	return s, nil
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

// GET /config/<uuid> -> 实时转换订阅为 sing-box 配置
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

	// 实时转换
	conv, err := converter.New(s.conf.TemplateFile)
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
