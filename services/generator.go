package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/follow1123/sing-box-ctl/singbox"
)

type Generator struct {
	log         logger.Logger
	singbox     *singbox.SingBox
	confUpdater ConfigUpdater
	server      *http.Server
}

func NewGenerator(log logger.Logger, singbox *singbox.SingBox, updater ConfigUpdater, conf config.GeneratorConfig) *Generator {

	mux := http.NewServeMux()
	g := &Generator{
		log:         log,
		singbox:     singbox,
		confUpdater: updater,
		server: &http.Server{
			Addr:    conf.Addr,
			Handler: mux,
		},
	}
	mux.Handle("/config", g)
	return g
}

func (g *Generator) Start() error {
	// 启动 HTTP 服务器
	g.log.Info("start generator on %s ...", g.server.Addr)
	if err := g.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("start server error: \n\t%w", err)
	}
	return nil
}

func (g *Generator) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := g.server.Shutdown(ctx); err != nil {
		g.log.Fatal(fmt.Errorf("Server Shutdown error: %v", err))
	}
	g.log.Info("generator terminated")
}

func (g *Generator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	mode := params.Get("mode")
	webui := params.Get("webui")
	android := params.Get("android")

	var optMode singbox.ProxyMode
	var optWebui bool
	var optAndroid bool
	if mode == "" {
		optMode = singbox.Mixed
	} else {
		proxyMode, err := singbox.StrToProxyMode(mode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		optMode = proxyMode
	}
	if webui != "" {
		optWebui = true
	}
	if android != "" {
		optAndroid = true
	}
	sc, err := g.confUpdater.LoadConfig()
	if err != nil {
		g.log.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data, err := g.singbox.ConvertConfig(sc, singbox.WithMode(optMode), singbox.WithWebUI(optWebui), singbox.WithAndroid(optAndroid))
	if err != nil {
		g.log.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	g.log.Info("response data")
	_, err = w.Write(data)
	if err != nil {
		g.log.Error(fmt.Errorf("write data error: \n\t%v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
