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
	U "github.com/follow1123/sing-box-ctl/updater"
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
	dataPath string
	server   *http.Server
	port     uint16
}

func New(port uint16) (*HttpShare, error) {
	conf, err := config.Default()
	if err != nil {
		return nil, err
	}
	h := &HttpShare{
		dataPath: conf.SingBoxConfigPath(),
		port:     port,
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
	updater, err := U.New(h.dataPath)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	var actions []U.Action
	// clash_api 相关配置
	if params.Has(paramWebuiAddr) {
		act := U.NewWebUIAddressAction()
		act.SetValue(params.Get(paramWebuiAddr))
		actions = append(actions, act)
	}
	if params.Has(paramWebuiSecret) {
		act := U.NewWebUISecretAction()
		act.SetValue(strings.TrimSpace(params.Get(paramWebuiSecret)))
		actions = append(actions, act)
	}
	if params.Has(paramResetWebui) {
		act := U.NewWebUIStatusAction()
		act.SetValue(true)
		actions = append(actions, act)
	}
	if params.Has(paramDisableWebui) {
		act := U.NewWebUIStatusAction()
		act.SetValue(false)
		actions = append(actions, act)
	}
	// inbound 模式相关配置
	if params.Has(paramMixedPort) {
		mixedPort, err := strconv.ParseInt(params.Get(paramMixedPort), 10, 16)
		if err != nil {
			handleBadRequestError(w, paramMixedPort, err)
			return
		}
		act := U.NewMixedPortAction()
		act.SetValue(mixedPort)
		actions = append(actions, act)
	}
	if params.Has(paramMixedEnableSystemProxy) {
		act := U.NewMixedSysProxyAction()
		act.SetValue(true)
		actions = append(actions, act)
	}
	if params.Has(paramMixedDisableSystemProxy) {
		act := U.NewMixedSysProxyAction()
		act.SetValue(false)
		actions = append(actions, act)
	}
	if params.Has(paramMixedAllowLAN) {
		act := U.NewMixedAllowLANAction()
		act.SetValue(true)
		actions = append(actions, act)
	}
	if params.Has(paramMixedDenyLAN) {
		act := U.NewMixedAllowLANAction()
		act.SetValue(false)
		actions = append(actions, act)
	}
	if params.Has(paramMixedMode) {
		actions = append(actions, U.NewMixedModeAction())
	}
	if params.Has(paramTunMode) {
		actions = append(actions, U.NewTunModeAction())
	}
	if params.Has(paramWindows) {
		act := U.NewPlatformAction()
		act.SetValue(U.PlatformWindows)
		actions = append(actions, act)
	}
	if params.Has(paramLinux) {
		act := U.NewPlatformAction()
		act.SetValue(U.PlatformLinux)
		actions = append(actions, act)
	}
	if params.Has(paramAndroid) {
		act := U.NewPlatformAction()
		act.SetValue(U.PlatformAndroid)
		actions = append(actions, act)
		actions = append(actions, U.NewTunModeAction())
	}
	// 修改配置
	if err := updater.Update(actions, params.Has(paramFormat)); err != nil {
		handleInternalServerError(w, err)
		return
	}
	// 写出配置
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(updater.Data()); err != nil {
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
