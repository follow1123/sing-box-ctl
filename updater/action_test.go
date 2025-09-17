package updater_test

import (
	"bytes"
	"encoding/json"
	"testing"

	JH "github.com/follow1123/sing-box-ctl/jsonhandler"
	"github.com/follow1123/sing-box-ctl/updater"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyConflicts(t *testing.T) {
	assert.True(t, updater.ActWebUIStatus.ConflictsWith(updater.ActWebUIAddress))
	assert.True(t, updater.ActWebUIAddress.ConflictsWith(updater.ActWebUIStatus))
	assert.True(t, updater.ActWebUISecret.ConflictsWith(updater.ActWebUIStatus))

	assert.True(t, updater.ActMixedMode.ConflictsWith(updater.ActMixedPort))
	assert.True(t, updater.ActMixedPort.ConflictsWith(updater.ActMixedMode))
	assert.True(t, updater.ActMixedSysProxy.ConflictsWith(updater.ActMixedMode))
	assert.True(t, updater.ActMixedAllowLAN.ConflictsWith(updater.ActMixedMode))

	assert.True(t, updater.ActTunMode.ConflictsWith(updater.ActTunMode))
	assert.True(t, updater.ActPlatForm.ConflictsWith(updater.ActPlatForm))
}

func TestWebUIStatus(t *testing.T) {
	data := []byte(`
	{
		"experimental": {
			"clash_api": {
				"external_controller": "127.0.0.1:9091",
				"external_ui": "./ui/",
				"external_ui_download_detour": "节点选择",
				"secret": "",
				"default_mode": "rule"
			}
		}
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("invalid value", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewWebUIStatusAction()
		act.SetValue("aaa")
		require.ErrorContains(t, act.Update(jh), "invalid value type")
	})
	t.Run("reset webui", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewWebUIStatusAction()
		act.SetValue(true)
		require.NoError(t, act.Update(jh))
		require.NoError(t, jh.Compact())
		require.NotEqual(t, data, jh.Data())
	})
	t.Run("disbled webui", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewWebUIStatusAction()
		act.SetValue(false)
		require.NoError(t, act.Update(jh))
		require.NoError(t, jh.Compact())
		require.Equal(t, []byte(`{"experimental":{}}`), jh.Data())
	})
	t.Run("assert enabled", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewWebUIStatusAction()
		require.True(t, act.IsEnabled(jh))
	})
	t.Run("assert disabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"experimental":{}}`))
		require.NoError(t, err)
		act := updater.NewWebUIStatusAction()
		require.False(t, act.IsEnabled(jh))
	})
}

func TestWebUIAddress(t *testing.T) {
	data := []byte(`
	{
		"experimental": {
			"clash_api": {
				"external_controller": "127.0.0.1:9091",
				"external_ui": "./ui/",
				"external_ui_download_detour": "节点选择",
				"secret": "",
				"default_mode": "rule"
			}
		}
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("get address when web ui is enabled", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewWebUIAddressAction()
		addr, err := act.GetAddress(jh)
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1:9091", addr)
	})
	t.Run("get address when web ui is disabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"experimental":{}}`))
		require.NoError(t, err)
		act := updater.NewWebUIAddressAction()
		_, err = act.GetAddress(jh)
		require.ErrorContains(t, err, "not exists")
	})
	t.Run("set address when web ui is enabled", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewWebUIAddressAction()
		value := "127.0.0.1:9090"
		act.SetValue(value)
		require.NoError(t, act.Update(jh))
		address, err := act.GetAddress(jh)
		require.NoError(t, err)
		require.Equal(t, value, address)
	})
	t.Run("set address when web ui is disabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"experimental":{}}`))
		require.NoError(t, err)
		act := updater.NewWebUIAddressAction()
		value := "127.0.0.1:9090"
		act.SetValue(value)
		require.NoError(t, act.Update(jh))
		address, err := act.GetAddress(jh)
		require.NoError(t, err)
		require.Equal(t, value, address)
	})
}

func TestWebUISecret(t *testing.T) {
	data := []byte(`
	{
		"experimental": {
			"clash_api": {
				"external_controller": "127.0.0.1:9091",
				"external_ui": "./ui/",
				"external_ui_download_detour": "节点选择",
				"secret": "",
				"default_mode": "rule"
			}
		}
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("get secret when web ui is enabled", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewWebUISecretAction()
		secret, err := act.GetSecret(jh)
		require.NoError(t, err)
		require.Equal(t, "", secret)
	})
	t.Run("get secret when web ui is disabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"experimental":{}}`))
		require.NoError(t, err)
		act := updater.NewWebUISecretAction()
		_, err = act.GetSecret(jh)
		require.ErrorContains(t, err, "not exists")
	})
	t.Run("set secret when web ui is enabled", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewWebUISecretAction()
		value := "12345"
		act.SetValue(value)
		require.NoError(t, act.Update(jh))
		secret, err := act.GetSecret(jh)
		require.NoError(t, err)
		require.Equal(t, value, secret)
	})
	t.Run("set secret when web ui is disabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"experimental":{}}`))
		require.NoError(t, err)
		act := updater.NewWebUISecretAction()
		value := "12345"
		act.SetValue(value)
		require.NoError(t, act.Update(jh))
		secret, err := act.GetSecret(jh)
		require.NoError(t, err)
		require.Equal(t, value, secret)
	})
}

func TestMixedMode(t *testing.T) {
	data := []byte(`
	{
		"inbounds": [
			{
				"type": "mixed",
				"tag": "mixed-in",
				"listen": "127.0.0.1",
				"listen_port": 7899,
				"set_system_proxy": false
			}
		]
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("no inbound when check status", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[]}`))
		require.NoError(t, err)
		act := updater.NewMixedModeAction()
		_, err = act.IsEnabled(jh)
		require.ErrorContains(t, err, "no inbound")
	})
	t.Run("no inbound type when check status", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{}]}`))
		require.NoError(t, err)
		act := updater.NewMixedModeAction()
		_, err = act.IsEnabled(jh)
		require.ErrorContains(t, err, "no inbound type")
	})
	t.Run("assert enabled", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewMixedModeAction()
		isEnabled, err := act.IsEnabled(jh)
		require.NoError(t, err)
		require.True(t, isEnabled)
	})
	t.Run("assert disabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"tun"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedModeAction()
		isEnabled, err := act.IsEnabled(jh)
		require.NoError(t, err)
		require.False(t, isEnabled)
	})
	t.Run("enable mixed mode", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[]}`))
		require.NoError(t, err)
		act := updater.NewMixedModeAction()
		require.NoError(t, act.Update(jh))
		isEnabled, err := act.IsEnabled(jh)
		require.NoError(t, err)
		require.True(t, isEnabled)
	})
}

func TestMixedPort(t *testing.T) {
	data := []byte(`
	{
		"inbounds": [
			{
				"type": "mixed",
				"tag": "mixed-in",
				"listen": "127.0.0.1",
				"listen_port": 7899,
				"set_system_proxy": false
			}
		]
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("get mixed port when mixed mode is not enabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"tun"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedPortAction()
		_, err = act.GetPort(jh)
		require.ErrorContains(t, err, "not enabled")
	})
	t.Run("get mixed port when field not exists", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"mixed"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedPortAction()
		_, err = act.GetPort(jh)
		require.ErrorContains(t, err, "not exists")
	})
	t.Run("get mixed port", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewMixedPortAction()
		port, err := act.GetPort(jh)
		require.NoError(t, err)
		require.Equal(t, uint16(7899), port)
	})
	t.Run("set mixed port", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"tun"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedPortAction()
		value := 7887
		act.SetValue(value)
		require.NoError(t, act.Update(jh))
		mixedModeAct := updater.NewMixedModeAction()
		isMixedModeEnabled, err := mixedModeAct.IsEnabled(jh)
		require.NoError(t, err)
		require.True(t, isMixedModeEnabled)
		port, err := act.GetPort(jh)
		require.NoError(t, err)
		require.Equal(t, uint16(value), port)
	})
}

func TestMixedSysProxy(t *testing.T) {
	data := []byte(`
	{
		"inbounds": [
			{
				"type": "mixed",
				"tag": "mixed-in",
				"listen": "127.0.0.1",
				"listen_port": 7899,
				"set_system_proxy": false
			}
		]
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("get mixed system proxy when mixed mode is not enabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"tun"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedSysProxyAction()
		_, err = act.IsSysProxyEnabled(jh)
		require.ErrorContains(t, err, "not enabled")
	})
	t.Run("get system proxy when field not exists", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"mixed"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedSysProxyAction()
		_, err = act.IsSysProxyEnabled(jh)
		require.ErrorContains(t, err, "not exists")
	})
	t.Run("get mixed system proxy", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewMixedSysProxyAction()
		isEnabled, err := act.IsSysProxyEnabled(jh)
		require.NoError(t, err)
		require.Equal(t, false, isEnabled)
	})
	t.Run("set mixed system proxy", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"tun"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedSysProxyAction()
		value := true
		act.SetValue(value)
		require.NoError(t, act.Update(jh))
		mixedModeAct := updater.NewMixedModeAction()
		isMixedModeEnabled, err := mixedModeAct.IsEnabled(jh)
		require.NoError(t, err)
		require.True(t, isMixedModeEnabled)
		isSysProxyEnabled, err := act.IsSysProxyEnabled(jh)
		require.NoError(t, err)
		require.Equal(t, value, isSysProxyEnabled)
	})
}

func TestMixedAllowLAN(t *testing.T) {
	data := []byte(`
	{
		"inbounds": [
			{
				"type": "mixed",
				"tag": "mixed-in",
				"listen": "127.0.0.1",
				"listen_port": 7899,
				"set_system_proxy": false
			}
		]
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("invalid value", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewMixedAllowLANAction()
		act.SetValue("aaa")
		require.ErrorContains(t, act.Update(jh), "invalid value type")
	})
	t.Run("get mixed allow lan when mixed mode is not enabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"tun"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedAllowLANAction()
		_, err = act.IsAllowLAN(jh)
		require.ErrorContains(t, err, "not enabled")
	})
	t.Run("get allow lan when field not exists", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"mixed"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedAllowLANAction()
		_, err = act.IsAllowLAN(jh)
		require.ErrorContains(t, err, "not exists")
	})
	t.Run("get mixed allow lan", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewMixedAllowLANAction()
		isAllowLAN, err := act.IsAllowLAN(jh)
		require.NoError(t, err)
		require.Equal(t, false, isAllowLAN)
	})
	t.Run("set mixed allow lan", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"tun"}]}`))
		require.NoError(t, err)
		act := updater.NewMixedAllowLANAction()
		value := true
		act.SetValue(value)
		require.NoError(t, act.Update(jh))
		mixedModeAct := updater.NewMixedModeAction()
		isMixedModeEnabled, err := mixedModeAct.IsEnabled(jh)
		require.NoError(t, err)
		require.True(t, isMixedModeEnabled)
		isAllowLAN, err := act.IsAllowLAN(jh)
		require.NoError(t, err)
		require.Equal(t, value, isAllowLAN)
	})
}

func TestTunMode(t *testing.T) {
	data := []byte(`
	{
		"inbounds": [
			{
				"type": "tun",
				"tag": "tun-in",
				"interface_name": "tun0",
				"address": "172.18.0.1/30",
				"mtu": 9000,
				"auto_route": true
			} 
		]
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("no inbound when check status", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[]}`))
		require.NoError(t, err)
		act := updater.NewTunModeAction()
		_, err = act.IsEnabled(jh)
		require.ErrorContains(t, err, "no inbound")
	})
	t.Run("no inbound type when check status", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{}]}`))
		require.NoError(t, err)
		act := updater.NewTunModeAction()
		_, err = act.IsEnabled(jh)
		require.ErrorContains(t, err, "no inbound type")
	})
	t.Run("assert enabled", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewTunModeAction()
		isEnabled, err := act.IsEnabled(jh)
		require.NoError(t, err)
		require.True(t, isEnabled)
	})
	t.Run("assert disabled", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[{"type":"mixed"}]}`))
		require.NoError(t, err)
		act := updater.NewTunModeAction()
		isEnabled, err := act.IsEnabled(jh)
		require.NoError(t, err)
		require.False(t, isEnabled)
	})
	t.Run("enable tun mode", func(t *testing.T) {
		jh, err := JH.FromData([]byte(`{"inbounds":[]}`))
		require.NoError(t, err)
		act := updater.NewTunModeAction()
		require.NoError(t, act.Update(jh))
		isEnabled, err := act.IsEnabled(jh)
		require.NoError(t, err)
		require.True(t, isEnabled)
	})
}

func TestPlatform(t *testing.T) {
	data := []byte(`
	{
		"inbounds": [
			{
				"type": "tun",
				"tag": "tun-in",
				"interface_name": "tun0",
				"address": "172.18.0.1/30",
				"mtu": 9000,
				"auto_route": true
			} 
		],
		"route": {}
	}
	`)
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, data))
	data = buf.Bytes()
	t.Run("set windows platform", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewPlatformAction()
		act.SetValue("windows")
		require.NoError(t, act.Update(jh))
		require.NoError(t, jh.Compact())
		require.Contains(t, string(jh.Data()), `"stack":"gvisor"`)
	})
	t.Run("set linux platform", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewPlatformAction()
		act.SetValue("linux")
		require.NoError(t, act.Update(jh))
		require.NoError(t, jh.Compact())
		require.Contains(t, string(jh.Data()), `"auto_redirect":true`)
	})
	t.Run("set android platform", func(t *testing.T) {
		jh, err := JH.FromData(data)
		require.NoError(t, err)
		act := updater.NewPlatformAction()
		act.SetValue("android")
		require.NoError(t, act.Update(jh))
		require.NoError(t, jh.Compact())
		result := string(jh.Data())
		require.Contains(t, result, `"override_android_vpn":true`)
		require.Contains(t, result, `"stack":"system"`)
	})
}
