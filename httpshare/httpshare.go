package httpshare

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/follow1123/sing-box-ctl/config"
	S "github.com/follow1123/sing-box-ctl/settings"
)

//go:embed index.html
var indexHtml string

const (
	paramWebuiStatus         = "webui_status"
	paramWebuiPort           = "webui_port"
	paramWebuiSecret         = "webui_secret"
	paramMixedStatus         = "mixed_status"
	paramMixedPort           = "mixed_port"
	paramMixedSysProxyStatus = "mixed_system_proxy_status"
	paramMixedShareStatus    = "mixed_share_status"
	paramTunStatus           = "tun_status"

	paramPlatform = "platform"

	paramFormat = "format"
)

const (
	homeUrlPath   = "/"
	configUrlPath = "/config"
)

type HttpShare struct {
	confPath     string
	tmplConfPath string
	server       *http.Server
	port         uint16
	localIP      string
}

func New(port uint16) (*HttpShare, error) {
	conf, err := config.Default()
	if err != nil {
		return nil, err
	}

	localIP, err := getLocalIP()
	if err != nil {
		return nil, err
	}

	h := &HttpShare{
		confPath:     conf.SingBox.ConfigFile,
		tmplConfPath: conf.SingBoxTemplateConfigFile,
		port:         port,
		localIP:      localIP,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(homeUrlPath, h.homeHandle)
	mux.HandleFunc(configUrlPath, h.configHandle)

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

func (h *HttpShare) Url() string {
	return fmt.Sprintf("http://%s:%d", h.localIP, h.port)
}

func (h *HttpShare) Stop(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *HttpShare) homeHandle(w http.ResponseWriter, r *http.Request) {
	s, err := S.NewSettings(h.confPath)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	tmplParam := make(map[string]any)
	tmplParam["ip"] = h.localIP
	tmplParam["port"] = h.port
	tmplParam["platform"] = string(S.PlatformWindows)

	webuiStatus, err := s.GetBool(S.StWebuiStatus)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	tmplParam[settingNameToStr(S.StWebuiStatus)] = webuiStatus
	if webuiStatus {
		webuiPort, err := s.GetUint16(S.StWebuiPort)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		tmplParam[settingNameToStr(S.StWebuiPort)] = webuiPort
		webuiSecret, err := s.GetString(S.StWebuiSecret)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		tmplParam[settingNameToStr(S.StWebuiSecret)] = webuiSecret
	}

	mixedStatus, err := s.GetBool(S.StMixedStatus)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}

	tmplParam[settingNameToStr(S.StMixedStatus)] = mixedStatus

	if mixedStatus {
		mixedPort, err := s.GetUint16(S.StMixedPort)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		tmplParam[settingNameToStr(S.StMixedPort)] = mixedPort
		mixedShare, err := s.GetBool(S.StMixedShareStatus)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		tmplParam[settingNameToStr(S.StMixedShareStatus)] = mixedShare

		mixedSysProxy, err := s.GetBool(S.StMixedSysProxyStatus)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		tmplParam[settingNameToStr(S.StMixedSysProxyStatus)] = mixedSysProxy

	}

	tunStatus, err := s.GetBool(S.StTunStatus)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
	tmplParam[settingNameToStr(S.StTunStatus)] = tunStatus

	jsonData, err := json.Marshal(tmplParam)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}

	tmpl := template.Must(template.New("home").Parse(indexHtml))
	if err := tmpl.Execute(w, string(jsonData)); err != nil {
		handleInternalServerError(w, err)
		return
	}
}

func (h *HttpShare) configHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}
	params := r.URL.Query()

	s, err := S.NewSettings(h.confPath)
	if err != nil {
		handleInternalServerError(w, err)
	}
	if err := s.SetTemplateConfig(h.tmplConfPath); err != nil {
		handleInternalServerError(w, err)
		return
	}
	settingsMap := make(map[S.SettingName]string)

	// clash_api 相关配置
	if params.Has(paramWebuiPort) {
		settingsMap[toSettingName(paramWebuiPort)] = params.Get(paramWebuiPort)
	}
	if params.Has(paramWebuiSecret) {
		settingsMap[toSettingName(paramWebuiSecret)] = params.Get(paramWebuiSecret)
	}
	if params.Has(paramWebuiStatus) {
		settingsMap[toSettingName(paramWebuiStatus)] = params.Get(paramWebuiStatus)
	}

	// inbound 模式相关配置
	if params.Has(paramMixedPort) {
		settingsMap[toSettingName(paramMixedPort)] = params.Get(paramMixedPort)
	}
	if params.Has(paramMixedSysProxyStatus) {
		settingsMap[toSettingName(paramMixedSysProxyStatus)] = params.Get(paramMixedSysProxyStatus)
	}
	if params.Has(paramMixedShareStatus) {
		settingsMap[toSettingName(paramMixedShareStatus)] = params.Get(paramMixedShareStatus)
	}
	if params.Has(paramMixedStatus) {
		settingsMap[toSettingName(paramMixedStatus)] = params.Get(paramMixedStatus)
	}

	if params.Has(paramTunStatus) {
		settingsMap[toSettingName(paramTunStatus)] = params.Get(paramTunStatus)
	}
	if params.Has(paramPlatform) {
		var err error
		switch S.Platform(params.Get(paramPlatform)) {
		case S.PlatformWindows:
			err = s.SetPlatform(S.PlatformWindows)
		case S.PlatformLinux:
			err = s.SetPlatform(S.PlatformLinux)
		case S.PlatformAndroid:
			// 安卓强制使用tun模式
			err = s.Set(S.StTunStatus, "true")
			if err == nil {
				err = s.SetPlatform(S.PlatformAndroid)
			}
		}
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
	}

	if err := s.SetMap(settingsMap); err != nil {
		handleInternalServerError(w, err)
		return
	}

	formatData := false
	if params.Has(paramFormat) {
		formatStr := params.Get(paramFormat)
		f, err := strconv.ParseBool(formatStr)
		if err != nil {
			// 请求参数错误，懒得多写一个处理方法了，无伤大雅
			handleInternalServerError(w, err)
			return
		}
		formatData = f
	}

	// 获取json数据
	data, err := s.ToJson(formatData)
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

func handleInternalServerError(w http.ResponseWriter, err error) {
	handleError(w, fmt.Errorf("Internal Server Error\n\t%w", err), http.StatusInternalServerError)
}

func toSettingName(paramName string) S.SettingName {
	return S.SettingName(strings.ReplaceAll(paramName, "_", "."))
}

func settingNameToStr(name S.SettingName) string {
	return strings.ReplaceAll(string(name), ".", "_")
}

func getLocalIP() (string, error) {
	// 连接到一个外部 IP（不需要真的发送数据）
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("test connect error:\n\t%w", err)
	}
	defer conn.Close()

	// 获取本地地址（本机出口 IP）
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil

}
