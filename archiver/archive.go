package archiver

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const latestName = "latest"

type Archive struct {
	dir      string
	name     string
	ext      string
	maxCount int
}

func NewArchive(dir string, name string, ext string, maxCount int) *Archive {
	return &Archive{
		dir:      dir,
		name:     name,
		ext:      ext,
		maxCount: maxCount,
	}
}

func (a *Archive) GetLatest() string {
	return a.fullPath(latestName)
}

func (a *Archive) Save(data []byte) error {
	archiveDir := filepath.Join(a.dir, a.name)
	des, err := os.ReadDir(archiveDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(archiveDir, 0755); err != nil {
			return fmt.Errorf("create archive %q error: \n\t%w", a.name, err)
		}
	} else if err != nil {
		return fmt.Errorf("check archive error: \n\t%w", err)
	}
	// 删除最旧的文件
	if des != nil && len(des) >= a.maxCount {
		var removeFileName string
		var oldestTime time.Time
		for _, e := range des {
			name := e.Name()[:len(e.Name())-len(a.ext)-1]
			if name == latestName {
				continue
			}
			um, err := strconv.ParseInt(name, 10, 64)
			if err != nil {
				return fmt.Errorf("parse unix micro error: \n\t%w", err)
			}

			modTime := time.UnixMicro(um)

			if removeFileName == "" || modTime.Before(oldestTime) {
				oldestTime = modTime
				removeFileName = name
			}
		}
		if removeFileName != "" {
			if err := os.Remove(a.fullPath(removeFileName)); err != nil {
				return fmt.Errorf("remove oldest file error: \n\t%w", err)
			}
		}
	}

	latestFile := a.GetLatest()
	fi, err := os.Stat(latestFile)
	if os.IsNotExist(err) {
		// 不存在直接创建
		if err := os.WriteFile(latestFile, data, 0664); err != nil {
			return fmt.Errorf("write latest file errro: \n\t%w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("check latest file error: \n\t%w", err)
	}

	// 重命名
	name := strconv.FormatInt(fi.ModTime().UnixMicro(), 10)
	if err := os.Rename(latestFile, a.fullPath(name)); err != nil {
		return fmt.Errorf("remove last file error: \n\t%w", err)
	}
	// 保持新文件
	if err := os.WriteFile(latestFile, data, 0664); err != nil {
		return fmt.Errorf("write latest file errro: \n\t%w", err)
	}

	return nil
}

func (a *Archive) fullPath(name string) string {
	return filepath.Join(a.dir, a.name, fullName(name, a.ext))

}
func fullName(name, ext string) string {
	return fmt.Sprintf("%s.%s", name, ext)
}
