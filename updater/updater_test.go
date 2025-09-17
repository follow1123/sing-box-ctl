package updater_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/follow1123/sing-box-ctl/config"
	JH "github.com/follow1123/sing-box-ctl/jsonhandler"
	"github.com/follow1123/sing-box-ctl/updater"
	"github.com/stretchr/testify/require"
)

var configData = []byte(`{
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:9090",
      "external_ui": "./ui/",
      "external_ui_download_detour": "节点选择",
      "secret": "",
      "default_mode": "rule"
    },
    "cache_file": {
      "enabled": true
    }
  },
  "inbounds": [
    {
      "type": "mixed",
      "tag": "mixed-in",
      "listen": "127.0.0.1",
      "listen_port": 7899,
      "set_system_proxy": true
    }
  ],
  "route": {
    "rules": [
    { "action": "sniff" },
    {
      "type": "logical",
      "mode": "or",
      "rules": [{ "protocol": "dns" }, { "port": 53 }],
      "action": "hijack-dns"
    },
    { "ip_is_private": true, "outbound": "直连" },
    { "clash_mode": "direct", "outbound": "直连" },
    { "clash_mode": "global", "outbound": "节点选择" }
    ],
    "default_domain_resolver": "dns-ali",
    "auto_detect_interface": true,
    "final": "漏网之鱼"
  }
}`)

func TestCreateSuccess(t *testing.T) {
	conf, err := config.New(t.TempDir())
	require.NoError(t, err)
	err = os.WriteFile(conf.SingBoxConfigPath(), configData, 0660)
	require.NoError(t, err)
	_, err = updater.New(conf.SingBoxConfigPath())
	require.NoError(t, err)
}

func TestCreateFailure(t *testing.T) {
	t.Run("config file not exists", func(t *testing.T) {
		conf, err := config.New(t.TempDir())
		require.NoError(t, err)
		_, err = updater.New(conf.SingBoxConfigPath())
		require.ErrorContains(t, err, "read json")
	})
	t.Run("config file is not json", func(t *testing.T) {
		conf, err := config.New(t.TempDir())
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(conf.SingBoxConfigPath(), []byte("234"), 0660))
		_, err = updater.New(conf.SingBoxConfigPath())
		require.ErrorContains(t, err, "not json")
	})
}

func TestRemoveConflacts(t *testing.T) {
	tests := []struct {
		testName     string
		actions      []updater.Action
		expectedKeys []updater.ActionKey
	}{
		{
			testName: "check web ui actions 1",
			actions: []updater.Action{
				updater.NewWebUIAddressAction(),
				updater.NewWebUISecretAction(),
				updater.NewWebUIStatusAction(),
			},
			expectedKeys: []updater.ActionKey{
				updater.ActWebUIStatus,
			},
		},
		{
			testName: "check web ui actions 2",
			actions: []updater.Action{
				updater.NewWebUIAddressAction(),
				updater.NewWebUIStatusAction(),
				updater.NewWebUISecretAction(),
			},
			expectedKeys: []updater.ActionKey{
				updater.ActWebUIAddress,
				updater.ActWebUISecret,
			},
		},
		{
			testName: "check duplicate actions 1",
			actions: []updater.Action{
				updater.NewWebUIAddressAction(),
				updater.NewWebUIAddressAction(),
			},
			expectedKeys: []updater.ActionKey{
				updater.ActWebUIAddress,
			},
		},
		{
			testName: "check duplicate actions 2",
			actions: []updater.Action{
				updater.NewTunModeAction(),
				updater.NewTunModeAction(),
			},
			expectedKeys: []updater.ActionKey{
				updater.ActTunMode,
			},
		},
		{
			testName: "check mixed mode actions 1",
			actions: []updater.Action{
				updater.NewMixedPortAction(),
				updater.NewMixedSysProxyAction(),
				updater.NewMixedModeAction(),
			},
			expectedKeys: []updater.ActionKey{
				updater.ActMixedMode,
			},
		},
		{
			testName: "check mixed mode actions 2",
			actions: []updater.Action{
				updater.NewMixedPortAction(),
				updater.NewMixedModeAction(),
				updater.NewMixedAllowLANAction(),
			},
			expectedKeys: []updater.ActionKey{
				updater.ActMixedPort,
				updater.ActMixedAllowLAN,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			result := updater.RemoveConflicts(test.actions)
			require.Equal(t, len(test.expectedKeys), len(result))
			for i := range result {
				require.Equal(t, test.expectedKeys[i], result[i].Key())
			}
		})
	}
}

func TestUpgradeSuccess(t *testing.T) {
	u, err := updater.FromData(configData)
	require.NoError(t, err)
	newData := []byte(`{
		"experimental": {
			"cache_file": {
				"enabled": true
			}
		},
		"inbounds": [
		],
		"outbounds": [
			{ "type": "direct", "tag": "直连" },
			{
			  "type": "selector",
			  "tag": "漏网之鱼",
			  "interrupt_exist_connections": false,
			  "outbounds": ["节点选择", "直连"],
			  "default": "节点选择"
			}
		],
		"route": {
			"rules": [
			{ "action": "sniff" },
			{
				"type": "logical",
				"mode": "or",
				"rules": [{ "protocol": "dns" }, { "port": 53 }],
				"action": "hijack-dns"
			},
			{ "ip_is_private": true, "outbound": "直连" },
			{ "clash_mode": "direct", "outbound": "直连" },
			{ "clash_mode": "global", "outbound": "节点选择" }
			],
			"default_domain_resolver": "dns-ali",
			"auto_detect_interface": true,
			"final": "漏网之鱼"
		}
	}`)
	err = u.Upgrade(newData, false)
	require.NoError(t, err)
	jh, err := JH.FromData(u.Data())
	require.NoError(t, err)
	address, err := updater.NewWebUIAddressAction().GetAddress(jh)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9090", address)
	require.Contains(t, string(u.Data()), "selector")
}

func TestUpgradeFailure(t *testing.T) {
	u, err := updater.FromData(bytes.Replace(configData, []byte("mixed"), []byte("socks"), 1))
	require.NoError(t, err)
	newData := []byte(`{
		"experimental": {
			"cache_file": {
				"enabled": true
			}
		},
		"inbounds": [
		],
		"outbounds": [
			{ "type": "direct", "tag": "直连" },
			{
			  "type": "selector",
			  "tag": "漏网之鱼",
			  "interrupt_exist_connections": false,
			  "outbounds": ["节点选择", "直连"],
			  "default": "节点选择"
			}
		],
		"route": {
			"rules": [
			{ "action": "sniff" },
			{
				"type": "logical",
				"mode": "or",
				"rules": [{ "protocol": "dns" }, { "port": 53 }],
				"action": "hijack-dns"
			},
			{ "ip_is_private": true, "outbound": "直连" },
			{ "clash_mode": "direct", "outbound": "直连" },
			{ "clash_mode": "global", "outbound": "节点选择" }
			],
			"default_domain_resolver": "dns-ali",
			"auto_detect_interface": true,
			"final": "漏网之鱼"
		}
	}`)
	err = u.Upgrade(newData, false)
	require.ErrorContains(t, err, "unsupported inbound type")
}

func TestSave(t *testing.T) {
	t.Run("update and save config", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, json.Compact(&buf, configData))
		conf, err := config.New(t.TempDir())
		require.NoError(t, err)
		err = os.WriteFile(conf.SingBoxConfigPath(), buf.Bytes(), 0660)
		require.NoError(t, err)
		u, err := updater.New(conf.SingBoxConfigPath())
		require.NoError(t, err)
		act := updater.NewWebUISecretAction()
		act.SetValue("12345")
		require.NoError(t, u.Update([]updater.Action{act}, false))
		require.True(t, u.IsModified())
	})
	t.Run("update same value do not save", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, json.Compact(&buf, configData))
		conf, err := config.New(t.TempDir())
		require.NoError(t, err)
		err = os.WriteFile(conf.SingBoxConfigPath(), buf.Bytes(), 0660)
		require.NoError(t, err)
		u, err := updater.New(conf.SingBoxConfigPath())
		require.NoError(t, err)
		act := updater.NewWebUIAddressAction()
		act.SetValue("127.0.0.1:9090")
		require.NoError(t, u.Update([]updater.Action{act}, false))
		require.False(t, u.IsModified())
	})
}
