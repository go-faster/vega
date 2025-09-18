package k8s

import (
	"sort"
	"strings"
)

// TODO: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/

// LabelApp is label for k8s app.
const LabelApp = "k8s-app"

const LabelNamespace = "io.kubernetes.pod.namespace"

const (
	LabelName      = "app.kubernetes.io/name"
	LabelPartOf    = "app.kubernetes.io/part-of"
	LabelCreatedBy = "app.kubernetes.io/created-by"
	LabelManagedBy = "app.kubernetes.io/managed-by"
)

// LabelSelector returns input for ListOptions.LabelSelector string.
func LabelSelector(m map[string]string) string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}
