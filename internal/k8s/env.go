package k8s

import (
	"sort"

	v1 "k8s.io/api/core/v1"
	apply "k8s.io/client-go/applyconfigurations/core/v1"
)

// Env returns environment variables list from map, sorting by name.
func Env(m map[string]string) []v1.EnvVar {
	var out []v1.EnvVar
	for k, v := range m {
		out = append(out, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name > out[j].Name
	})

	return out
}

// EnvApply returns environment variables list from map, sorting by name.
func EnvApply(m map[string]string) []apply.EnvVarApplyConfiguration {
	var out []apply.EnvVarApplyConfiguration
	for k, v := range m {
		k := k
		v := v
		out = append(out, apply.EnvVarApplyConfiguration{
			Name:  &k,
			Value: &v,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return *out[i].Name < *out[j].Name
	})

	return out
}
