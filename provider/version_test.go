package provider_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemplateVersionsAndRestore(t *testing.T) {
	p, _ := newProvider(t)
	uuid, err := p.AddTemplate("t")
	require.NoError(t, err)

	require.NoError(t, p.SaveTemplate(uuid, []byte("v0")))
	require.NoError(t, p.SaveTemplate(uuid, []byte("v1")))
	require.NoError(t, p.SaveTemplate(uuid, []byte("v2")))

	vers, err := p.TemplateVersions(uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current", "last", "old"}, vers)

	// 还原 old（v0），current 变为 v0，滚动保留一层
	require.NoError(t, p.RestoreTemplate(uuid, "old"))
	cur, err := p.ReadTemplate(uuid)
	require.NoError(t, err)
	require.Equal(t, "v0", string(cur))
	last, err := p.TemplateVersions(uuid)
	require.NoError(t, err)
	require.Len(t, last, 3, "还原后仍保留三层历史")

	require.Error(t, p.RestoreTemplate(uuid, "not-a-version"))
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
