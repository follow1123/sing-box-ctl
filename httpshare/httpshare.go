package httpshare

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/olekukonko/tablewriter"
)

const (
	paramDisableWebui = "disable_webui"
	paramResetWebui   = "reset_webui"
	paramWebuiAddr    = "webui_addr"
	paramWebuiSecret  = "webui_secret"

	paramMixedMode               = "mixed"
	paramMixedPort               = "mixed_port"
	paramMixedEnableSystemProxy  = "enable_sys_proxy"
	paramMixedDisableSystemProxy = "disable_sys_proxy"
	paramMixedAllowLAN           = "allow_lan"
	paramMixedDenyLAN            = "deny_lan"
	paramTunMode                 = "tun"

	paramWindows = "windows"
	paramLinux   = "linux"
	paramAndroid = "android"

	paramFormat = "format"
)

const urlPath = "/config"

type HttpShare struct {
	confPath     string
	tmplConfPath string
	server       *http.Server
	port         uint16
}

func New(port uint16) (*HttpShare, error) {
	conf, err := config.Default()
	if err != nil {
		return nil, err
	}
	h := &HttpShare{
		confPath:     conf.SingBoxConfigPath(),
		tmplConfPath: conf.SingBoxTmplConfigPath(),
		port:         port,
	}
	mux := http.NewServeMux()
	mux.HandleFunc(urlPath, h.handle)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	h.server = server
	return h, nil
}

func (h *HttpShare) Share() error {
	if err := h.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("start server error:\n\t%w", err)
	}
	return nil
}

func (h *HttpShare) PrintHelp() error {
	table := tablewriter.NewTable(os.Stdout, tablewriter.WithEastAsian(false))
	paramsInfo := [][]string{
		{"Parameter", "Description"},
		{paramDisableWebui, "disable web ui"},
		{paramResetWebui, "reset web ui"},
		{paramWebuiAddr, "type string, set web ui address, (eg: localhost:9090)"},
		{paramWebuiSecret, "type string set web ui secret"},

		{paramMixedMode, "reset to mixed mode"},
		{paramMixedPort, "type int, set mixed mode port"},
		{paramMixedEnableSystemProxy, "enable system proxy in mixed mode"},
		{paramMixedDisableSystemProxy, "disable system proxy in mixed mode"},
		{paramMixedAllowLAN, "allow LAN Sharing"},
		{paramMixedDenyLAN, "deny LAN Sharing"},
		{paramTunMode, "reset to tun mode"},

		{paramWindows, "use windows platform config"},
		{paramLinux, "use linux platform config"},
		{paramAndroid, "use android platform config"},

		{paramFormat, "format config"},
	}

	table.Header(paramsInfo[0])
	if err := table.Bulk(paramsInfo[1:]); err != nil {
		log.Fatal(fmt.Errorf("render param info error:\n\t%w", err))
	}

	if err := table.Render(); err != nil {
		log.Fatal(fmt.Errorf("render param info error:\n\t%w", err))
	}
	return nil
}

func (h *HttpShare) Url() string {
	return fmt.Sprintf("http://localhost:%d%s", h.port, urlPath)
}

func (h *HttpShare) Open() error {
	return nil
}

func (h *HttpShare) Stop(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *HttpShare) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}
	params := r.URL.Query()

	s, err := settings.NewSettings(h.tmplConfPath, h.confPath)
	if err != nil {
		handleInternalServerError(w, err)
	}
	// clash_api 相关配置
	webuiSetting := settings.WebUISettings{Enabled: true}
	if params.Has(paramWebuiAddr) {
		weuiAddr := params.Get(paramWebuiAddr)
		webuiSetting.Addr = &weuiAddr
	}
	if params.Has(paramWebuiSecret) {
		webuiSecret := strings.TrimSpace(params.Get(paramWebuiSecret))
		webuiSetting.Password = &webuiSecret
	}
	if params.Has(paramResetWebui) {
		webuiSetting.Reset = true
	}
	if params.Has(paramDisableWebui) {
		webuiSetting.Enabled = false
	}
	s.UpdateWebUISettings(webuiSetting)

	// inbound 模式相关配置
	mixedProxySettings := settings.MixedProxySettings{Enabled: true}
	if params.Has(paramMixedPort) {
		mixedPort, err := strconv.ParseInt(params.Get(paramMixedPort), 10, 16)
		if err != nil {
			handleBadRequestError(w, paramMixedPort, err)
			return
		}
		mp := uint16(mixedPort)
		mixedProxySettings.Port = &mp
	}
	if params.Has(paramMixedEnableSystemProxy) {
		var enable = true
		mixedProxySettings.EnableSystemProxy = &enable
	}
	if params.Has(paramMixedDisableSystemProxy) {
		var enable = false
		mixedProxySettings.EnableSystemProxy = &enable
	}
	if params.Has(paramMixedAllowLAN) {
		var enable = true
		mixedProxySettings.AllowLAN = &enable
	}
	if params.Has(paramMixedDenyLAN) {
		var enable = false
		mixedProxySettings.AllowLAN = &enable
	}
	if params.Has(paramMixedMode) {
		mixedProxySettings.Reset = true
	}
	if err := s.UpdateMixedProxySettings(mixedProxySettings); err != nil {
		handleInternalServerError(w, err)
		return
	}

	if params.Has(paramTunMode) {
		s.UpdateTunSettings(true)
	}
	if params.Has(paramWindows) {
		if err := s.UpdatePlatformSettings(settings.PlatformWindows); err != nil {
			handleInternalServerError(w, err)
			return
		}

	}
	if params.Has(paramLinux) {
		if err := s.UpdatePlatformSettings(settings.PlatformLinux); err != nil {
			handleInternalServerError(w, err)
			return
		}

	}
	if params.Has(paramAndroid) {
		fmt.Printf("\"use android\": %v\n", "use android")
		s.UpdateTunSettings(true)
		if err := s.UpdatePlatformSettings(settings.PlatformAndroid); err != nil {
			handleInternalServerError(w, err)
			return
		}

	}
	// 获取json数据
	data, err := s.ToJson(params.Has(paramFormat))
	if err != nil {
		handleInternalServerError(w, err)
		return
	}

	// 写出配置
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		handleInternalServerError(w, err)
		return
	}
}

func handleError(w http.ResponseWriter, err error, code int) {
	log.Print(err)
	http.Error(w, err.Error(), code)
}

func handleBadRequestError(w http.ResponseWriter, paramName string, err error) {
	handleError(w, fmt.Errorf("invalid param '%s'\n\t%w", paramName, err), http.StatusBadRequest)
}

func handleInternalServerError(w http.ResponseWriter, err error) {
	handleError(w, fmt.Errorf("Internal Server Error\n\t%w", err), http.StatusInternalServerError)
}
