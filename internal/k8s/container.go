package k8s

import (
	"strings"
)

type ContainerURI struct {
	ID     string
	Schema string
}

func ContainerID(uri string) ContainerURI {
	const delim = "://"
	idx := strings.Index(uri, delim)
	if idx == 0 || len(uri) <= (idx+len(delim)) {
		return ContainerURI{
			ID: uri,
		}
	}
	return ContainerURI{
		ID:     uri[idx+len(delim):],
		Schema: uri[:idx],
	}
}
