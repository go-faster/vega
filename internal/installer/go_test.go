package installer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoBuild_Run(t *testing.T) {
	t.Cleanup(func() {
		_ = os.RemoveAll("_out/bin")
	})
	g := GoBuild{
		Binary: "test-binary",
	}

	require.NoError(t, g.Run(context.Background()))

	stat, err := os.Stat("_out/bin/test-binary")
	require.NoError(t, err)
	assert.True(t, stat.Mode().IsRegular())
	assert.NotEqual(t, 0, stat.Size())
}
