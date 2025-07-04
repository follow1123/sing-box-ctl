package services

import (
	"errors"
	"time"

	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/follow1123/sing-box-ctl/singbox"
)

type ConfigUpdater interface {
	LoadConfig() (singbox.SubscriptionConfig, error)
	UpdateConfig() (singbox.SubscriptionConfig, error)
	NeedUpdate(lastUpdateTime *time.Time) bool
	GetLastUpdateTime() (*time.Time, error)
}

type Updater struct {
	log            logger.Logger
	singbox        singbox.SingBox
	confUpdater    ConfigUpdater
	lastUpdateTime *time.Time

	ticker *time.Ticker
	done   chan struct{}
}

func NewUpdater(log logger.Logger, singbox *singbox.SingBox, cu ConfigUpdater) (*Updater, error) {
	lastUpdateTime, err := cu.GetLastUpdateTime()
	if err != nil {
		return nil, err
	}

	return &Updater{
		log:            log,
		singbox:        *singbox,
		confUpdater:    cu,
		lastUpdateTime: lastUpdateTime,
	}, nil
}

func (u *Updater) Start() {
	if u.ticker != nil {
		u.log.Fatal(errors.New("ticker already started"))
	}
	u.log.Info("start updater")
	u.ticker = time.NewTicker(time.Minute)
	u.done = make(chan struct{})
	for {
		select {
		case <-u.done:
			u.log.Info("updater terminated")
		case <-u.ticker.C:
			if u.confUpdater.NeedUpdate(u.lastUpdateTime) {
				if err := u.Update(true); err != nil {
					u.log.Error(err)
				}
			}
		}
	}
}

func (u *Updater) Stop() {
	if u.ticker == nil {
		u.log.Fatal(errors.New("ticker not start"))
	}
	u.done <- struct{}{}
	u.ticker.Stop()
}

func (u *Updater) Update(fetchRemote bool, opts ...singbox.Opt) error {
	u.log.Debug("run update")

	var subConfig singbox.SubscriptionConfig
	if fetchRemote {
		u.log.Info("update remote configuration")
		remoteConfig, err := u.confUpdater.UpdateConfig()
		if err != nil {
			return err
		}
		subConfig = remoteConfig
		// 更新上传获取时间
		lastTime, err := u.confUpdater.GetLastUpdateTime()
		if err != nil {
			return err
		}
		u.lastUpdateTime = lastTime
	} else {
		localConfig, err := u.confUpdater.LoadConfig()
		if err != nil {
			return err
		}

		subConfig = localConfig
	}
	data, err := u.singbox.ConvertConfig(subConfig, opts...)
	if err != nil {
		return err
	}

	if err := u.singbox.SaveConfig(data); err != nil {
		return err
	}

	if err := u.singbox.Restart(); err != nil {
		return err
	}

	return nil
}
