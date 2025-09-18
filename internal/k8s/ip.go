package k8s

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"strconv"
)

// URL returns http url to port in minikube.
func URL(port int, parts ...string) string {
	u := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(LocalDomain, strconv.Itoa(port)),
	}
	for _, p := range parts {
		u.Path = path.Join(u.Path, p)
	}
	return u.String()
}

// Node ports of common local k8s services.
const (
	PortAPI      = 30800
	PortRegistry = 30080
	PortHostOTEL = 30090
)

// ServiceHost for kubernetes service.
func ServiceHost(name string, namespace ...string) string {
	ns := Namespace
	for _, v := range namespace {
		ns = v
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", name, ns)
}

func ServiceAddr(name string, port int, namespace ...string) string {
	return net.JoinHostPort(ServiceHost(name, namespace...), strconv.Itoa(port))
}

func Service(schema, name string, port int, p ...string) string {
	u := &url.URL{
		Scheme: schema,
		Host:   net.JoinHostPort(ServiceHost(name), strconv.Itoa(port)),
	}
	for _, v := range p {
		u.Path = path.Join(u.Path, v)
	}
	return u.String()
}

const (
	ServicePortHTTP    = 8080
	ServicePortMetrics = 9000
	SchemaHTTP         = "http"
)

func ServiceHTTP(name string, p ...string) string {
	return Service(SchemaHTTP, name, ServicePortHTTP, p...)
}
