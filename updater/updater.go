package updater

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"

	JH "github.com/follow1123/sing-box-ctl/jsonhandler"
)

type Updater struct {
	path        string
	sum         []byte
	jsonHandler *JH.JsonHandler
}

func FromData(data []byte) (*Updater, error) {
	jsonHandler, err := JH.FromData(data)
	if err != nil {
		return nil, err
	}
	return &Updater{
		jsonHandler: jsonHandler,
	}, nil
}

func New(path string) (*Updater, error) {
	jsonHandler, err := JH.FromFile(path)
	if err != nil {
		return nil, err
	}
	configSum := md5.Sum(jsonHandler.Data())
	return &Updater{
		path:        path,
		sum:         configSum[:],
		jsonHandler: jsonHandler,
	}, nil
}

func (u *Updater) Update(actions []Action, format bool) error {
	actions = RemoveConflicts(actions)
	var platformAction Action
	for _, act := range actions {
		// 如果有 platform action 留到最后执行
		if act.Key() == ActPlatForm {
			platformAction = act
			continue
		}
		if err := act.Update(u.jsonHandler); err != nil {
			return err
		}
	}
	if platformAction == nil {
		platformAction = NewPlatformAction()
	}
	if err := platformAction.Update(u.jsonHandler); err != nil {
		return err
	}
	var err error
	if format {
		err = u.jsonHandler.Format()
	} else {
		err = u.jsonHandler.Compact()
	}
	return err
}

func (u *Updater) Upgrade(data []byte, format bool) error {
	var actions []Action
	webUIStatusAct := NewWebUIStatusAction()
	if webUIStatusAct.IsEnabled(u.jsonHandler) {
		webUIAddrAct := NewWebUIAddressAction()
		addr, err := webUIAddrAct.GetAddress(u.jsonHandler)
		if err != nil {
			return err
		}
		webUIAddrAct.SetValue(addr)
		actions = append(actions, webUIAddrAct)
		webUISecretAct := NewWebUISecretAction()
		secret, err := webUISecretAct.GetSecret(u.jsonHandler)
		if err != nil {
			return err
		}
		webUISecretAct.SetValue(secret)
		actions = append(actions, webUISecretAct)
	} else {
		webUIStatusAct.SetValue(false)
		actions = append(actions, webUIStatusAct)
	}
	inboundType, exists := u.jsonHandler.GetString("inbounds.0.type")
	if !exists {
		return errors.New("no inbound or no inbound type")
	}
	switch inboundType {
	case "mixed":
		mixedPortAct := NewMixedPortAction()
		port, err := mixedPortAct.GetPort(u.jsonHandler)
		if err != nil {
			return err
		}
		mixedPortAct.SetValue(port)
		actions = append(actions, mixedPortAct)
		mixedSysProxyAct := NewMixedSysProxyAction()
		isSysProxyEnabled, err := mixedSysProxyAct.IsSysProxyEnabled(u.jsonHandler)
		if err != nil {
			return err
		}
		mixedSysProxyAct.SetValue(isSysProxyEnabled)
		actions = append(actions, mixedSysProxyAct)
		mixedAllowLANAct := NewMixedAllowLANAction()
		isAllowLAN, err := mixedAllowLANAct.IsAllowLAN(u.jsonHandler)
		if err != nil {
			return err
		}
		mixedAllowLANAct.SetValue(isAllowLAN)
		actions = append(actions, mixedAllowLANAct)
	case "tun":
		actions = append(actions, NewTunModeAction())
	default:
		return fmt.Errorf("unsupported inbound type '%s'", inboundType)
	}
	jsonHandler, err := JH.FromData(data)
	if err != nil {
		return err
	}
	u.jsonHandler = jsonHandler
	return u.Update(actions, format)
}

func (u *Updater) Data() []byte {
	return u.jsonHandler.Data()
}
func (u *Updater) IsModified() bool {
	sum := md5.Sum(u.Data())
	return !bytes.Equal(sum[:], u.sum)
}

func (u *Updater) Save() error {
	return u.jsonHandler.SaveTo(u.path)
}

func RemoveConflicts(actions []Action) []Action {
	removeIdx := make(map[int]struct{})
	for i := len(actions) - 1; i >= 0; i-- {
		if _, removed := removeIdx[i]; removed {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			if _, removed := removeIdx[j]; removed {
				continue
			}

			if actions[i].Key().ConflictsWith(actions[j].Key()) {
				removeIdx[j] = struct{}{}
			}
		}
	}
	var result []Action
	for i, action := range actions {
		if _, removed := removeIdx[i]; !removed {
			result = append(result, action)
		}
	}
	return result
}

func (u *Updater) JsonHandler() *JH.JsonHandler {
	return u.jsonHandler
}
