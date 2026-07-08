package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTopPodsSortsByCPU(t *testing.T) {
	raw := []byte(`{"items":[{"metadata":{"namespace":"default","name":"a"},"containers":[{"usage":{"cpu":"10m","memory":"64Mi"}}]},{"metadata":{"namespace":"default","name":"b"},"containers":[{"usage":{"cpu":"250m","memory":"128Mi"}}]}]}`)
	items, err := parseTopPods(raw, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "b", items[0].Pod)
	assert.Equal(t, int64(250), items[0].CPUMilli)
	assert.Greater(t, items[0].MemoryBytes, int64(0))
}

func TestParseTopNodesAppliesLimit(t *testing.T) {
	raw := []byte(`{"items":[{"metadata":{"name":"n1"},"usage":{"cpu":"100m","memory":"1Gi"}},{"metadata":{"name":"n2"},"usage":{"cpu":"500m","memory":"2Gi"}}]}`)
	items, err := parseTopNodes(raw, 1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "n2", items[0].Node)
}
