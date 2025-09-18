package k8s

// Credentials for local development.
const (
	Name       = "vega"     // platform common Name, default local credential
	longSecret = "password" // platform long secret, when Name is too short

	// OAuth2 mock config.

	AuthClientID     = Name
	AuthClientSecret = Name

	SecretAPIOAuth     = "vega.api.oauth"
	SecretGrafanaOAuth = "vega.grafana.oauth"

	// Minio config.

	S3ID     = Name
	S3Secret = longSecret
	S3Region = "us-east-1" // default region
	S3Bucket = Name

	// PostgreSQL config.

	DBPassword = Name
	DBUser     = Name
	DBName     = Name
	DB         = "host=psql.vega.svc.cluster.local port=5432 user=vega dbname=vega password=vega sslmode=disable"
	DBURL      = "postgres://vega:vega@psql.vega.svc.cluster.local:5432/vega?sslmode=disable"

	// K8S config.

	Namespace       = Name
	NamespaceSystem = "kube-system"

	UserGitea      = 1000
	UserClickhouse = 2000

	LocalDomain = "kube.local"

	ConfigName = "vega.cfg"

	FieldManager = "vega-install"

	ServiceAuth              = "auth"
	ServiceAPI               = "api"
	ServiceImageBuilder      = "image-builder"
	ServiceDeployer          = "deployer"
	ServiceKafka             = "kafka"
	ServiceLogTailer         = "log-tailer"
	ServiceLogSaver          = "log-saver"
	ServicePostgres          = "psql"
	ServiceRedis             = "redis"
	ServiceClickhouse        = "clickhouse"
	ServiceS3                = "s3"
	ServiceGitea             = "gitea"
	ServiceGitLab            = "gitlab"
	ServiceTeamCity          = "teamcity"
	ServicePrometheus        = "prometheus"
	ServiceGrafana           = "grafana"
	ServiceJaeger            = "jaeger"
	ServiceOTEL              = "otel"
	ServiceChrome            = "chrome"
	ServiceDebug             = "debug"
	ServiceKubeStateMetrics  = "kube-state"
	ServiceClickhouseOTEL    = "ch-otel"
	ServiceClickhouseCluster = "ch-cluster"

	PortClickhouse = 9000
	PortKafka      = 9092
	PortOTELHTTP   = 4318
	PortOTELGRPC   = 4317

	Init = "install-init"

	ServiceRegistry = "registry" // from addon, in system ns
)

const (
	GitLabDebugRootPassword = "9a3bd550-f534-4356-8dcd-393df51c7cd1"
	GitLabDebugRootLogin    = "root"
)

// Secret data keys.
const (
	DataToken = "token"
	DataCA    = "ca.crt"
)
