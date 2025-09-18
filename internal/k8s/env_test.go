package k8s

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvApply(t *testing.T) {
	data, err := json.Marshal(EnvApply(map[string]string{
		"FOO": "BAR",
		"BAZ": "FOO",
		"ONE": "TWO",
	}))
	require.NoError(t, err)
	require.JSONEq(t, `[
	{"name":  "BAZ", "value":  "FOO"},
	{"name":  "FOO", "value":  "BAR"},
	{"name":  "ONE", "value":  "TWO"}
]`, string(data))
}
