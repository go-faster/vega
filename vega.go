// Package vega implements PaaS R&D.
package vega

// Injected environment variables from k8s to application.
const (
	EnvNode = "VEGA_K8S_NODE"
	EnvPod  = "VEGA_K8S_POD"
	EnvNS   = "VEGA_K8S_NS"

	EnvProject     = "VEGA_PROJECT"
	EnvApplication = "VEGA_APP"
	EnvUnit        = "VEGA_UNIT"
	EnvEnvironment = "VEGA_ENV"
	EnvNamespace   = "VEGA_NS"
)

// Possible values for vega environment.
const (
	EnvProduction  = "prod"
	EnvStaging     = "stage"
	EnvDevelopment = "dev"
	EnvTesting     = "test"
)

// Deployment labels.
const (
	PlatformPrefix = "vega."

	LabelApplication = PlatformPrefix + "app"
	LabelCommit      = PlatformPrefix + "commit"
	LabelUnit        = PlatformPrefix + "unit"
	LabelProject     = PlatformPrefix + "project"
	LabelBaseImage   = PlatformPrefix + "base"
	LabelPrometheus  = PlatformPrefix + "prometheus"
	LabelEnvironment = PlatformPrefix + "env"
	LabelNamespace   = PlatformPrefix + "ns"
	LabelDataCenter  = PlatformPrefix + "dc"
	LabelCluster     = PlatformPrefix + "cluster"
	LabelRegion      = PlatformPrefix + "region"
	LabelIngress     = PlatformPrefix + "ingress"
)

// Secret labels.
const (
	LabelGitLabInstance = PlatformPrefix + "gitlab.instance"
	LabelSecretType     = PlatformPrefix + "secret.type"
)

// Secret types.
const (
	SecretTypeGitLabAccessToken             = "gitlab-access-token"
	SecretTypeGitLabRunnerRegistrationToken = "gitlab-runner-registration-token" //#nosec G101
	SecretTypeGitLabApplicationCredentials  = "gitlab-application-credentials"   //#nosec G101

	SecretTypeRemoteToken = "remote-token" // opaque token for remote cluster, created on host cluster
	SecretTypeHostToken   = "host-token"   // service account token, created on remote cluster
)

const (
	AnnotationComponent  = PlatformPrefix + "component"
	AnnotationSecretName = PlatformPrefix + "secret.name"

	AnnotationGitLabInstance    = PlatformPrefix + "gitlab.instance"
	AnnotationGitLabHost        = PlatformPrefix + "gitlab.host"
	AnnotationGitLabURL         = PlatformPrefix + "gitlab.url"
	AnnotationGitLabInternalURL = PlatformPrefix + "gitlab.internal.url"
	AnnotationGitLabName        = PlatformPrefix + "gitlab.name"

	AnnotationRemoteName       = PlatformPrefix + "remote.name"
	AnnotationRemoteRegion     = PlatformPrefix + "remote.region"
	AnnotationRemoteEnv        = PlatformPrefix + "remote.env"
	AnnotationRemoteDataCenter = PlatformPrefix + "remote.dc"
	AnnotationRemoteAddress    = PlatformPrefix + "remote.address"
	AnnotationRemoteHost       = PlatformPrefix + "remote.host"
	AnnotationRemotePort       = PlatformPrefix + "remote.port"
	AnnotationRemoteDomain     = PlatformPrefix + "remote.domain"
	AnnotationRemoteRegistry   = PlatformPrefix + "remote.registry"
)

const (
	ComponentSecrets = "secrets"
)

const (
	EnvClickHouseAddr        = "VEGA_CLICKHOUSE_ADDR"
	EnvClickHouseUser        = "VEGA_CLICKHOUSE_USER"
	EnvClickHousePassword    = "VEGA_CLICKHOUSE_PASSWORD"
	EnvClickHouseDB          = "VEGA_CLICKHOUSE_DB"
	EnvClickHouseLogsTable   = "VEGA_CLICKHOUSE_LOGS_TABLE"
	EnvClickHouseTracesTable = "VEGA_CLICKHOUSE_TRACES_TABLE"

	EnvVaultAddr = "VAULT_ADDR"
	EnvVaultRole = "VEGA_VAULT_ROLE"

	EnvAdminList = "VEGA_ADMIN_LIST"

	EnvClickHouseCA   = "VEGA_CLICKHOUSE_CA"          // issuing CA
	EnvClickHouseCert = "VEGA_CLICKHOUSE_CERT"        // certificate file
	EnvClickHouseKey  = "VEGA_CLICKHOUSE_PRIVATE_KEY" // private key file

	EnvKafkaAddr     = "VEGA_KAFKA_ADDR"
	EnvKafkaTopic    = "VEGA_KAFKA_TOPIC"
	EnvKafkaUser     = "VEGA_KAFKA_USER"
	EnvKafkaPassword = "VEGA_KAFKA_PASSWORD"
	EnvKafkaBalancer = "VEGA_KAFKA_BALANCER"

	EnvLogLevel = "OTEL_LOG_LEVEL"

	EnvOTLPAddr = "VEGA_OTLP_ADDR"

	EnvListenAddr = "LISTEN_ADDR"

	EnvRootURL = "ROOT_URL" // for grafana
)

// Istio variables
const (
	IstioNamespace       = "vega-istio"
	IstioWorkloadLabel   = "istio-ingressgateway-app"
	IstioHTTPGatewayPort = 80
)
