package archiver_test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/follow1123/sing-box-ctl/archiver"
	"github.com/stretchr/testify/require"
)

const data = `{
	"name": "1234",
	"age": 18
}`

func TestSaveArchiveFileSucceed(t *testing.T) {
	r := require.New(t)
	var dir = t.TempDir()
	var name = "test-file"
	var maxCount = 3
	archive := archiver.NewArchive(dir, name, "json", maxCount)

	var saveCount = 10

	for i := range saveCount {
		t.Run(fmt.Sprintf("save-%d", i), func(t *testing.T) {
			err := archive.Save([]byte(data))
			r.NoError(err, "save error")
			time.Sleep(5 * time.Millisecond)
			fi, err := os.Stat(archive.GetLatest())
			t.Logf("latest file mod time: %v", fi.ModTime().UnixMicro())
			r.NoError(err, "check latest file error")
			des, err := os.ReadDir(filepath.Join(dir, name))
			for idx, e := range des {
				t.Logf("file: %d-%s", idx, e.Name())
			}
			r.NoError(err, "check archived file count error")
			count := int(math.Min(float64(i+1), float64(maxCount)))
			r.Equal(count, len(des))
		})
	}
}
