package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPodLogOptionsSupportsPreviousSinceAndTailLines(t *testing.T) {
	tail := int64(50)
	opts, err := buildPodLogOptions(map[string]interface{}{"sinceMinutes": 15}, tail, true, "app")
	require.NoError(t, err)
	assert.Equal(t, int64(50), *opts.TailLines)
	assert.True(t, opts.Previous)
	assert.Equal(t, "app", opts.Container)
	require.NotNil(t, opts.SinceSeconds)
	assert.Equal(t, int64(900), *opts.SinceSeconds)
}

func TestBuildPodLogOptionsSinceRFC3339OverridesSinceMinutes(t *testing.T) {
	tail := int64(100)
	opts, err := buildPodLogOptions(map[string]interface{}{"sinceMinutes": 15, "since": "2026-07-08T08:00:00Z"}, tail, false, "")
	require.NoError(t, err)
	assert.Nil(t, opts.SinceSeconds)
	require.NotNil(t, opts.SinceTime)
	assert.Equal(t, "2026-07-08T08:00:00Z", opts.SinceTime.Time.UTC().Format("2006-01-02T15:04:05Z"))
}

func TestBuildPodLogOptionsRejectsInvalidSince(t *testing.T) {
	tail := int64(100)
	_, err := buildPodLogOptions(map[string]interface{}{"since": "not-a-time"}, tail, false, "")
	require.Error(t, err)
}
