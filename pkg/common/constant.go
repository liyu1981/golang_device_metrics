package common

const (
	EnvKeyGoEnv string = "GO_ENV"

	EnvKeyRunIntegrationTests string = "RUN_INTEGRATION_TESTS"

	EnvKeyIOTDBType string = "IOT_DB_TYPE"
	EnvKeyIOTDbPath string = "IOT_DB_PATH"

	EnvKeyIOTHttpHostPort string = "IOT_HTTP_HOST_PORT"
	EnvKeyIOTGrpcHostPort string = "IOT_GRPC_HOST_PORT"

	EnvKeyIOTDefaultRate  string = "IOT_DEFAULT_RATE"
	EnvKeyIOTDefaultBurst string = "IOT_DEFAULT_BURST"

	LoggerNameIOTCore       string = "iot_core"
	LoggerNameRestfulServer string = "restful_server"
	LoggerNameGrpcServer    string = "grpc_server"
	LoggerFieldIOTCategory  string = "category"
	LoggerCategoryIOTMetric string = "metric"
	LoggerCategoryIOTAlert  string = "alert"
	LoggerCategoryIOTConfig string = "config"
)
