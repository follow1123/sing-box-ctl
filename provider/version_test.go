package provider_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemplateVersionsAndRestore(t *testing.T) {
	tm, _ := newTm(t)
	uuid, err := tm.AddTemplate("t")
	require.NoError(t, err)

	put := func(v int) []byte {
		return []byte(fmt.Sprintf(`{"v": %d}`, v))
	}
	require.NoError(t, tm.SaveTemplate(uuid, put(0)))
	require.NoError(t, tm.SaveTemplate(uuid, put(1)))
	require.NoError(t, tm.SaveTemplate(uuid, put(2)))

	vers, err := tm.TemplateVersions(uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current", "last", "old"}, vers)

	// 还原 old（v0），current 变为 v0，滚动保留一层
	require.NoError(t, tm.RestoreTemplate(uuid, "old"))
	cur, err := tm.ReadTemplate(uuid)
	require.NoError(t, err)
	require.Equal(t, normJSON(t, string(put(0))), normJSON(t, string(cur)))
	vers, err = tm.TemplateVersions(uuid)
	require.NoError(t, err)
	require.Len(t, vers, 3, "还原后仍保留三层历史")

	require.Error(t, tm.RestoreTemplate(uuid, "not-a-version"))
}

func TestSubscriptionVersionsAndRestore(t *testing.T) {
	pm, _ := newManager(t)
	uuid, err := pm.Add("sub", "", "upload", "")
	require.NoError(t, err)

	require.NoError(t, pm.SaveNodes(uuid, nodesJSON(t, "节点-a")))
	vers, err := pm.SubscriptionVersions(uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current"}, vers)

	require.NoError(t, pm.SaveNodes(uuid, nodesJSON(t, "节点-b")))
	require.NoError(t, pm.SaveNodes(uuid, nodesJSON(t, "节点-c")))
	vers, err = pm.SubscriptionVersions(uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current", "last", "old"}, vers)

	require.NoError(t, pm.RestoreSubscription(uuid, "old"))
	data, err := pm.ReadSubscription(uuid)
	require.NoError(t, err)
	var nodes []map[string]any
	require.NoError(t, json.Unmarshal(data, &nodes))
	require.Equal(t, "节点-a", nodes[0]["tag"])
}
