package clash

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/follow1123/sing-box-ctl/archiver"
	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/follow1123/sing-box-ctl/singbox"
)

type ClashUpdater struct {
	log     logger.Logger
	archive *archiver.Archive

	url      string
	interval int64
}

func NewClashUpdater(log logger.Logger, confHome string, subcriptionUrl string, intervalStr string) (*ClashUpdater, error) {
	// 检查 url
	_, err := url.Parse(subcriptionUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid http url: %q, error: \n\t%w", subcriptionUrl, err)
	}

	interval, err := parseInterval(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("parse interval config error: \n\t%w", err)
	}
	log.Debug("update interval: %d second", interval)

	return &ClashUpdater{
		log:      log,
		url:      subcriptionUrl,
		interval: interval,
		archive:  archiver.NewArchive(confHome, "clash-subscription-config", "yaml", 3),
	}, nil
}

func (cu *ClashUpdater) LoadConfig() (singbox.SubscriptionConfig, error) {
	data, err := os.ReadFile(cu.archive.GetLatest())
	if err != nil {
		return nil, fmt.Errorf("read local clash config error: \n\t%w", err)
	}
	return NewClashConfig(data), nil
}

func (cu *ClashUpdater) UpdateConfig() (singbox.SubscriptionConfig, error) {
	data, err := downloadConfig(cu.url)
	if err != nil {
		return nil, err
	}
	if err := cu.archive.Save(data); err != nil {
		return nil, err
	}
	return NewClashConfig(data), nil
}

func (cu *ClashUpdater) NeedUpdate(lastUpdateTime *time.Time) bool {
	if lastUpdateTime == nil {
		return true
	}
	second := time.Since(*lastUpdateTime).Seconds()
	cu.log.Debug("last time: %v, interval: %v, update interval: %v", lastUpdateTime.Format("2006-01-02 15:04:05"), second, cu.interval)
	return second > float64(cu.interval)
}

func (cu *ClashUpdater) GetLastUpdateTime() (*time.Time, error) {
	fi, err := os.Stat(cu.archive.GetLatest())
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("init last updated time error \n\t%w", err)
	}
	modTime := fi.ModTime()
	return &modTime, nil
}

func downloadConfig(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %q error: %w", url, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}
	return data, nil
}

func parseInterval(intervalConf string) (int64, error) {
	intervalStr := intervalConf[:len(intervalConf)-1]
	intervalInt, err := strconv.ParseInt(intervalStr, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("parse interval error: \n\t%w", err)
	}
	if intervalInt < 0 {
		return 0, fmt.Errorf("interval cannot be negative: %d", intervalInt)
	}

	unit := intervalConf[len(intervalConf)-1:]
	switch unit {
	case "m":
		if intervalInt < 1 {
			return 60, nil
		}
		return intervalInt * 60, nil
	case "h":
		return intervalInt * 3600, nil
	case "d":
		return intervalInt * 3600 * 24, nil
	default:
		return 0, fmt.Errorf("invalid unit %q, available units: [m|h|d]", unit)
	}
}
