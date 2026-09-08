package provider_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/follow1123/sing-box-ctl/provider"
	"github.com/stretchr/testify/require"
)

// normJSON 把 JSON 文本归一为紧凑单行，用于内容比较（模板存取会规范缩进）
func normJSON(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, []byte(s)))
	return buf.String()
}

// newTm 构造模板数据管理器（working_dir/templates/data.json）
func newTm(t *testing.T) (*provider.TemplateManager, string) {
	t.Helper()
	workingDir := t.TempDir()
	tm, err := provider.NewTemplateManager(workingDir)
	require.NoError(t, err)
	return tm, workingDir
}

func TestAddTemplateDuplicateName(t *testing.T) {
	tm, _ := newTm(t)
	_, err := tm.AddTemplate("dup")
	require.NoError(t, err)

	_, err = tm.AddTemplate("dup")
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate template name")
}

func TestListTemplatesOrder(t *testing.T) {
	tm, _ := newTm(t)

	oldUUID, err := tm.AddTemplate("旧模板")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	newUUID, err := tm.AddTemplate("新模板")
	require.NoError(t, err)

	// 最新创建的在前
	list := tm.ListTemplates()
	require.Len(t, list, 2)
	require.Equal(t, newUUID, list[0].Uuid, "最新创建应在最前")
	require.Equal(t, oldUUID, list[1].Uuid)
	require.False(t, list[0].Default)
	require.False(t, list[1].Default)

	// 设为默认后默认模板排最前
	require.NoError(t, tm.SetDefaultTemplate(oldUUID))
	list = tm.ListTemplates()
	require.True(t, list[0].Default)
	require.Equal(t, oldUUID, list[0].Uuid)
}

func TestTemplateSaveReadAndPersist(t *testing.T) {
	tm, workingDir := newTm(t)
	uuid, err := tm.AddTemplate("t")
	require.NoError(t, err)

	// 未保存时无内容
	_, err = tm.ReadTemplate(uuid)
	require.ErrorContains(t, err, "no content")

	content := `{"inbounds": [], "outbounds": []}`
	require.NoError(t, tm.SaveTemplate(uuid, []byte(content)))
	data, err := tm.ReadTemplate(uuid)
	require.NoError(t, err)
	require.Equal(t, normJSON(t, content), normJSON(t, string(data)))

	// 立即落盘后重新加载仍能读到（含默认与排序字段）
	require.NoError(t, tm.SetDefaultTemplate(uuid))
	require.NoError(t, tm.SaveNow())

	tm2, err := provider.NewTemplateManager(workingDir)
	require.NoError(t, err)
	info := tm2.GetTemplate(uuid)
	require.NotNil(t, info)
	require.Equal(t, "t", info.Name)
	require.True(t, info.Default)
	data, err = tm2.ReadTemplate(uuid)
	require.NoError(t, err)
	require.Equal(t, normJSON(t, content), normJSON(t, string(data)))

	require.FileExists(t, filepath.Join(workingDir, "templates", "data.json"))
}

// TestTemplateMigrateLegacy 历史散目录（templates/<uuid>/ 的 name/current/last/old/default）
// 应一次性导入 templates/data.json，创建时间取 name 文件 mtime
func TestTemplateMigrateLegacy(t *testing.T) {
	workingDir := t.TempDir()
	dir := filepath.Join(workingDir, "templates", "legacy-tpl")
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "name"), []byte("旧模板"), 0600))
	// 默认标记
	require.NoError(t, os.WriteFile(filepath.Join(dir, "default"), nil, 0600))
	v0 := `{"v": 0}`
	v1 := `{"v": 1}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "current"), []byte(v1), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "last"), []byte(v0), 0600))

	// 手工调旧 name 文件时间（验证 created_at 带入排序）
	past := time.Now().Add(-24 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "name"), past, past))

	tm, err := provider.NewTemplateManager(workingDir)
	require.NoError(t, err)
	require.NoError(t, err)

	list := tm.ListTemplates()
	require.Len(t, list, 1)
	require.Equal(t, "旧模板", list[0].Name)
	require.True(t, list[0].Default, "default 文件应迁移为默认标记")

	// 内容版本与顺序（current/last -> 0/1）
	vers, err := tm.TemplateVersions(list[0].Uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current", "last"}, vers)
	data, err := tm.ReadTemplate(list[0].Uuid)
	require.NoError(t, err)
	require.Equal(t, normJSON(t, v1), normJSON(t, string(data)))

	// 默认模板接口直接返回该 uuid
	def, err := tm.DefaultTemplate()
	require.NoError(t, err)
	require.Equal(t, list[0].Uuid, def)

	// data.json 已生成
	require.FileExists(t, filepath.Join(workingDir, "templates", "data.json"))
}
