package archiver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
)

var MaxCount = 3

type Archiver struct {
	archiveDir string
}

func New(archiveDir string) (*Archiver, error) {
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, fmt.Errorf("init archive directory error:\n\t%w", err)
	}
	return &Archiver{archiveDir: archiveDir}, nil
}

func (a *Archiver) Save(data []byte) error {
	// 1. 判断 config.ArchiveDir 内是否有 MaxCount 个文件，有就删除最旧的一个文件
	files := a.GetSortedFiles()
	if len(files) >= MaxCount {
		filesToDelete := files[MaxCount-1:]
		for _, f := range filesToDelete {
			if err := os.Remove(f); err != nil {
				return fmt.Errorf("remove '%s' error\n\t%w", f, err)
			}
		}
	}
	// 2. 保存文件
	name := strings.ReplaceAll(uuid.New().String(), "-", "")
	latestFile := filepath.Join(a.archiveDir, name)
	if err := os.WriteFile(latestFile, data, 0660); err != nil {
		return fmt.Errorf("save archive file error\n\t%w", err)
	}
	return nil
}

func (a *Archiver) GetLatest() string {
	files := a.GetSortedFiles()
	if len(files) == 0 {
		return ""
	}
	return files[0]
}

func (a *Archiver) GetSortedFiles() []string {
	des, _ := os.ReadDir(a.archiveDir)
	var files []string
	slices.SortFunc(des, func(a, b os.DirEntry) int {
		infoA, err := a.Info()
		if err != nil {
			log.Fatal(fmt.Errorf("sort archived file and check '%s' error:\n\t%w", a.Name(), err))
		}

		infoB, err := b.Info()
		if err != nil {
			log.Fatal(fmt.Errorf("sort archived file and check '%s' error:\n\t%w", b.Name(), err))
		}
		return int(infoB.ModTime().UnixNano() - infoA.ModTime().UnixNano())
	})
	for _, d := range des {
		files = append(files, filepath.Join(a.archiveDir, d.Name()))
	}
	return files
}
