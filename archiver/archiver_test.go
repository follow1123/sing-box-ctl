package archiver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/follow1123/sing-box-ctl/archiver"
	"github.com/stretchr/testify/require"
)

func newArchiver(t *testing.T) (*archiver.Archiver, string) {
	t.Helper()
	archiveDir := t.TempDir()
	a, err := archiver.New(archiveDir)
	require.NoError(t, err)
	return a, archiveDir
}

func TestInit(t *testing.T) {
	archiveDir := t.TempDir()
	_, err := archiver.New(archiveDir)
	require.NoError(t, err)
	info, err := os.Stat(archiveDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestGetSortedFiles(t *testing.T) {
	a, archiveDir := newArchiver(t)

	require.Equal(t, 0, len(a.GetSortedFiles()))

	// 写入 5M 数据，防止创建时间相同
	data := make([]byte, 5*1024*1024)
	fileA := filepath.Join(archiveDir, "a.txt")
	fileB := filepath.Join(archiveDir, "b.txt")
	fileC := filepath.Join(archiveDir, "c.txt")
	require.NoError(t, os.WriteFile(fileA, data, 0660))
	require.NoError(t, os.WriteFile(fileB, data, 0660))
	require.NoError(t, os.WriteFile(fileC, data, 0660))

	files := a.GetSortedFiles()
	require.Equal(t, 3, len(files))
	require.Equal(t, fileC, files[0])
	require.Equal(t, fileB, files[1])
	require.Equal(t, fileA, files[2])
}

func TestGetLatest(t *testing.T) {
	a, archiveDir := newArchiver(t)

	require.Equal(t, "", a.GetLatest())

	// 写入 5M 数据，防止创建时间相同
	data := make([]byte, 5*1024*1024)
	fileA := filepath.Join(archiveDir, "a.txt")
	fileB := filepath.Join(archiveDir, "b.txt")
	fileC := filepath.Join(archiveDir, "c.txt")
	require.NoError(t, os.WriteFile(fileA, data, 0660))
	require.NoError(t, os.WriteFile(fileB, data, 0660))
	require.NoError(t, os.WriteFile(fileC, data, 0660))

	require.Equal(t, fileC, a.GetLatest())
}

func TestSave(t *testing.T) {
	a, _ := newArchiver(t)

	// 写入 5M 数据，防止创建时间相同
	data := make([]byte, 5*1024*1024)
	require.NoError(t, a.Save(data))
	require.Equal(t, 1, len(a.GetSortedFiles()))
	require.NoError(t, a.Save(data))
	require.Equal(t, 2, len(a.GetSortedFiles()))
	require.NoError(t, a.Save(data))
	require.Equal(t, 3, len(a.GetSortedFiles()))
	require.NoError(t, a.Save(data))
	require.Equal(t, 3, len(a.GetSortedFiles()))
	require.NoError(t, a.Save(data))
	require.Equal(t, 3, len(a.GetSortedFiles()))
}
