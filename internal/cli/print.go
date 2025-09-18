package cli

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
)

// Print object as yaml.
func Print(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("!<err>:%v", err)
	}
	out, err := yaml.JSONToYAML(data)
	if err != nil {
		return fmt.Sprintf("!<err>:%v", err)
	}
	return string(out)
}
