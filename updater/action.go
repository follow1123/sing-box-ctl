package updater

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	JH "github.com/follow1123/sing-box-ctl/jsonhandler"
)

type ActionKey string

const (
	ActWebUIStatus   ActionKey = "webui.status"
	ActWebUIAddress  ActionKey = "webui.address"
	ActWebUISecret   ActionKey = "webui.secret"
	ActMixedMode     ActionKey = "mode.mixed"
	ActMixedPort     ActionKey = "mode.mixed.port"
	ActMixedSysProxy ActionKey = "mode.mixed.sysproxy"
	ActMixedAllowLAN ActionKey = "mode.mixed.allowlan"
	ActTunMode       ActionKey = "mode.tun"
	ActPlatForm      ActionKey = "platform"
)

func (a ActionKey) ConflictsWith(other ActionKey) bool {
	switch a {
	case ActWebUIStatus:
		return other == ActWebUIStatus || other == ActWebUIAddress || other == ActWebUISecret
	case ActWebUIAddress:
		return other == ActWebUIStatus || other == ActWebUIAddress
	case ActWebUISecret:
		return other == ActWebUIStatus || other == ActWebUISecret
	case ActMixedMode:
		return other == ActMixedMode || other == ActMixedPort || other == ActMixedSysProxy || other == ActMixedAllowLAN || other == ActTunMode
	case ActMixedPort:
		return other == ActMixedMode || other == ActMixedPort || other == ActTunMode
	case ActMixedSysProxy:
		return other == ActMixedMode || other == ActMixedSysProxy || other == ActTunMode
	case ActMixedAllowLAN:
		return other == ActMixedMode || other == ActMixedAllowLAN || other == ActTunMode
	case ActTunMode:
		return other == ActTunMode || other == ActMixedMode || other == ActMixedPort || other == ActMixedSysProxy || other == ActMixedAllowLAN
	case ActPlatForm:
		return other == ActPlatForm
	}
	return false
}

type Action interface {
	Key() ActionKey
	Get(jsonHandler *JH.JsonHandler) (any, error)
	SetValue(value any)
	Update(jsonHandler *JH.JsonHandler) error
}

type BaseAction struct {
	key   ActionKey
	path  string
	value any
}

func (b *BaseAction) Key() ActionKey {
	return b.key
}

func (b *BaseAction) SetValue(value any) {
	b.value = value
}

type WebUIStatusAction struct {
	BaseAction
}

func NewWebUIStatusAction() *WebUIStatusAction {
	return &WebUIStatusAction{
		BaseAction: BaseAction{
			key:  ActWebUIStatus,
			path: "experimental.clash_api",
		},
	}
}

func (w *WebUIStatusAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	_, exists := jsonHandler.GetResult(w.path)
	return exists, nil
}

func (w *WebUIStatusAction) Update(jsonHandler *JH.JsonHandler) error {
	status, ok := w.value.(bool)
	if !ok {
		return errors.New("invalid value type, should be bool")
	}
	if status {
		return jsonHandler.SetRaw(w.path, []byte(`{
  "external_controller": "127.0.0.1:9090",
  "external_ui": "./ui/",
  "external_ui_download_detour": "节点选择",
  "secret": "",
  "default_mode": "rule"
}`))
	} else {
		return jsonHandler.Delete(w.path)
	}
}

func (w *WebUIStatusAction) IsEnabled(jsonHandler *JH.JsonHandler) bool {
	value, _ := w.Get(jsonHandler)
	return value.(bool)
}

type WebUIAddressAction struct {
	BaseAction
}

func NewWebUIAddressAction() *WebUIAddressAction {
	return &WebUIAddressAction{
		BaseAction: BaseAction{
			key:  ActWebUIAddress,
			path: "experimental.clash_api.external_controller",
		},
	}
}

func (w *WebUIAddressAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	address, exists := jsonHandler.GetString(w.path)
	if !exists {
		return nil, fmt.Errorf("'%s' not exists", w.path)
	}
	return address, nil
}

func (w *WebUIAddressAction) Update(jsonHandler *JH.JsonHandler) error {
	statusAct := NewWebUIStatusAction()
	if !statusAct.IsEnabled(jsonHandler) {
		statusAct.SetValue(true)
		if err := statusAct.Update(jsonHandler); err != nil {
			return err
		}
	}
	return jsonHandler.Set(w.path, w.value)
}

func (w *WebUIAddressAction) GetAddress(jsonHandler *JH.JsonHandler) (string, error) {
	value, err := w.Get(jsonHandler)
	if err != nil {
		return "", err
	}
	return value.(string), nil
}

type WebUISecretAction struct {
	BaseAction
}

func NewWebUISecretAction() *WebUISecretAction {
	return &WebUISecretAction{
		BaseAction: BaseAction{
			key:  ActWebUISecret,
			path: "experimental.clash_api.secret",
		},
	}
}

func (w *WebUISecretAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	secret, exists := jsonHandler.GetString(w.path)
	if !exists {
		return nil, fmt.Errorf("'%s' not exists", w.path)
	}
	return secret, nil
}

func (w *WebUISecretAction) Update(jsonHandler *JH.JsonHandler) error {
	statusAct := NewWebUIStatusAction()
	if !statusAct.IsEnabled(jsonHandler) {
		statusAct.SetValue(true)
		if err := statusAct.Update(jsonHandler); err != nil {
			return err
		}
	}
	return jsonHandler.Set(w.path, w.value)
}

func (w *WebUISecretAction) GetSecret(jsonHandler *JH.JsonHandler) (string, error) {
	value, err := w.Get(jsonHandler)
	if err != nil {
		return "", err
	}
	return value.(string), nil
}

type MixedModeAction struct {
	BaseAction
}

func NewMixedModeAction() *MixedModeAction {
	return &MixedModeAction{
		BaseAction: BaseAction{
			key:  ActMixedMode,
			path: "inbounds.0",
		},
	}
}

func (w *MixedModeAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	inboundRoot, exists := jsonHandler.GetResult(w.path)
	if !exists {
		return nil, errors.New("no inbound")
	}
	typeResult := inboundRoot.Get("type")
	if !typeResult.Exists() {
		return nil, errors.New("no inbound type")
	}
	return typeResult.String() == "mixed", nil
}

func (w *MixedModeAction) Update(jsonHandler *JH.JsonHandler) error {
	return jsonHandler.SetRaw(w.path, []byte(`{
  "type": "mixed",
  "tag": "mixed-in",
  "listen": "127.0.0.1",
  "listen_port": 7899,
  "set_system_proxy": false
}`))
}

func (w *MixedModeAction) IsEnabled(jsonHandler *JH.JsonHandler) (bool, error) {
	value, err := w.Get(jsonHandler)
	if err != nil {
		return false, err
	}
	return value.(bool), nil
}

func checkIsMixedMode(jsonHandler *JH.JsonHandler) error {
	mixedAct := NewMixedModeAction()
	isEnabled, err := mixedAct.IsEnabled(jsonHandler)
	if err != nil {
		return err
	}
	if !isEnabled {
		return errors.New("mixed mode is not enabled")
	}
	return nil
}

type MixedPortAction struct {
	BaseAction
}

func NewMixedPortAction() *MixedPortAction {
	return &MixedPortAction{
		BaseAction: BaseAction{
			key:  ActMixedPort,
			path: "inbounds.0.listen_port",
		},
	}
}

func (w *MixedPortAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	if err := checkIsMixedMode(jsonHandler); err != nil {
		return nil, err
	}
	port, exists := jsonHandler.GetInt(w.path)
	if !exists {
		return nil, fmt.Errorf("'%s' not exists", w.path)
	}
	return port, nil
}

func (w *MixedPortAction) Update(jsonHandler *JH.JsonHandler) error {
	mixedAct := NewMixedModeAction()
	isEnabled, err := mixedAct.IsEnabled(jsonHandler)
	if err != nil || !isEnabled {
		if err := mixedAct.Update(jsonHandler); err != nil {
			return err
		}
	}
	return jsonHandler.Set(w.path, w.value)
}

func (w *MixedPortAction) GetPort(jsonHandler *JH.JsonHandler) (uint16, error) {
	value, err := w.Get(jsonHandler)
	if err != nil {
		return 0, err
	}
	return uint16(value.(int64)), nil
}

type MixedSysProxyAction struct {
	BaseAction
}

func NewMixedSysProxyAction() *MixedSysProxyAction {
	return &MixedSysProxyAction{
		BaseAction: BaseAction{
			key:  ActMixedSysProxy,
			path: "inbounds.0.set_system_proxy",
		},
	}
}

func (w *MixedSysProxyAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	if err := checkIsMixedMode(jsonHandler); err != nil {
		return nil, err
	}
	sysproxy, exists := jsonHandler.GetBool(w.path)

	if !exists {
		return nil, fmt.Errorf("'%s' not exists", w.path)
	}
	return sysproxy, nil
}

func (w *MixedSysProxyAction) Update(jsonHandler *JH.JsonHandler) error {
	mixedAct := NewMixedModeAction()
	isEnabled, err := mixedAct.IsEnabled(jsonHandler)
	if err != nil || !isEnabled {
		if err := mixedAct.Update(jsonHandler); err != nil {
			return err
		}
	}
	return jsonHandler.Set(w.path, w.value)
}

func (w *MixedSysProxyAction) IsSysProxyEnabled(jsonHandler *JH.JsonHandler) (bool, error) {
	value, err := w.Get(jsonHandler)
	if err != nil {
		return false, err
	}
	return value.(bool), nil
}

type MixedAllowLANAction struct {
	BaseAction
}

func NewMixedAllowLANAction() *MixedAllowLANAction {
	return &MixedAllowLANAction{
		BaseAction: BaseAction{
			key:  ActMixedAllowLAN,
			path: "inbounds.0.listen",
		},
	}
}

func (w *MixedAllowLANAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	if err := checkIsMixedMode(jsonHandler); err != nil {
		return nil, err
	}
	listenAddr, exists := jsonHandler.GetString(w.path)
	if !exists {
		return nil, fmt.Errorf("'%s' not exists", w.path)
	}
	return strings.Contains(listenAddr, "::"), nil
}

func (w *MixedAllowLANAction) Update(jsonHandler *JH.JsonHandler) error {
	mixedAct := NewMixedModeAction()
	isEnabled, err := mixedAct.IsEnabled(jsonHandler)
	if err != nil || !isEnabled {
		if err := mixedAct.Update(jsonHandler); err != nil {
			return err
		}
	}
	allowLAN, ok := w.value.(bool)
	if !ok {
		return errors.New("invalid value type, should be bool")
	}
	var listenAddr string
	if allowLAN {
		listenAddr = "::"
	} else {
		listenAddr = "127.0.0.1"
	}
	return jsonHandler.Set(w.path, listenAddr)
}

func (w *MixedAllowLANAction) IsAllowLAN(jsonHandler *JH.JsonHandler) (bool, error) {
	value, err := w.Get(jsonHandler)
	if err != nil {
		return false, err
	}
	return value.(bool), nil
}

type TunModeAction struct {
	BaseAction
}

func NewTunModeAction() *TunModeAction {
	return &TunModeAction{
		BaseAction: BaseAction{
			key:  ActTunMode,
			path: "inbounds.0",
		},
	}
}

func (w *TunModeAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	inboundRoot, exists := jsonHandler.GetResult(w.path)
	if !exists {
		return nil, errors.New("no inbound")
	}
	typeResult := inboundRoot.Get("type")
	if !typeResult.Exists() {
		return nil, errors.New("no inbound type")
	}
	return typeResult.String() == "tun", nil
}

func (w *TunModeAction) Update(jsonHandler *JH.JsonHandler) error {
	return jsonHandler.SetRaw(w.path, []byte(`{
  "type": "tun",
  "tag": "tun-in",
  "interface_name": "tun0",
  "address": "172.18.0.1/30",
  "mtu": 9000,
  "auto_route": true
}`))
}

func (w *TunModeAction) IsEnabled(jsonHandler *JH.JsonHandler) (bool, error) {
	value, err := w.Get(jsonHandler)
	if err != nil {
		return false, err
	}
	return value.(bool), nil
}

type Platform = string

const (
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
	PlatformAndroid Platform = "android"
)

type PlatformAction struct {
	BaseAction
}

func NewPlatformAction() *PlatformAction {
	return &PlatformAction{
		BaseAction: BaseAction{
			key:   ActPlatForm,
			value: runtime.GOOS,
		},
	}
}

func (w *PlatformAction) Get(jsonHandler *JH.JsonHandler) (any, error) {
	return w.value, nil
}

func (w *PlatformAction) Update(jsonHandler *JH.JsonHandler) error {
	platform := w.value.(Platform)
	tunAct := NewTunModeAction()
	isTunEnabled, err := tunAct.IsEnabled(jsonHandler)
	if err != nil {
		return err
	}
	switch platform {
	case PlatformWindows:
		if isTunEnabled {
			if err := jsonHandler.Set("inbounds.0.stack", "gvisor"); err != nil {
				return err
			}
		}
	case PlatformLinux:
		if isTunEnabled {
			if err := jsonHandler.Set("inbounds.0.auto_redirect", true); err != nil {
				return err
			}
		}
	case PlatformAndroid:
		if !isTunEnabled {
			return errors.New("android platform must use tun mode")
		}

		if err := jsonHandler.Set("route.override_android_vpn", true); err != nil {
			return err
		}

		if err := jsonHandler.Set("inbounds.0.stack", "system"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid platform: %s", platform)
	}
	return nil
}

func (w *PlatformAction) GetPlatform() Platform {
	return w.value.(Platform)
}
