package provider_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAddTemplateDuplicateName(t *testing.T) {
	p, _ := newProvider(t)
	_, err := p.AddTemplate("dup")
	require.NoError(t, err)

	_, err = p.AddTemplate("dup")
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate template name")
}

// 目录 mtime 用于排序，用 Chtimes 显式控制以便断言稳定
func TestListTemplatesOrder(t *testing.T) {
	p, _ := newProvider(t)

	oldUUID, err := p.AddTemplate("旧模板")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	newUUID, err := p.AddTemplate("新模板")
	require.NoError(t, err)

	// 新模板目录 mtime 更晚
	list, err := p.ListTemplates()
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, newUUID, list[0].Uuid, "最新创建应在最前")
	require.Equal(t, oldUUID, list[1].Uuid)

	// 设为默认后默认模板排最前
	require.NoError(t, p.SetDefaultTemplate(oldUUID))
	list, err = p.ListTemplates()
	require.NoError(t, err)
	require.True(t, list[0].Default)
	require.Equal(t, oldUUID, list[0].Uuid)
}

func TestTemplateModTimeFallback(t *testing.T) {
	p, workingDir := newProvider(t)
	uuid, err := p.AddTemplate("t")
	require.NoError(t, err)

	// 排序基于模板 name 文件 mtime：人为改早，验证其排后
	nameFile := filepath.Join(workingDir, "templates", uuid, "name")
	past := time.Now().Add(-24 * time.Hour)
	require.NoError(t, os.Chtimes(nameFile, past, past))

	_, err = p.AddTemplate("t2")
	require.NoError(t, err)
	list, err := p.ListTemplates()
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.NotEqual(t, uuid, list[0].Uuid, "name 文件更早的应排后")
}
