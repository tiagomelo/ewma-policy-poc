package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// Config holds all configuration needed by this app.
type Config struct {
	// metrics server.
	MetricsServerPort int `envconfig:"METRICS_SERVER_PORT" required:"true"`

	// Prometheus.
	PromTemplateFile     string `envconfig:"PROM_TEMPLATE_FILE" required:"true"`
	PromOutputFile       string `envconfig:"PROM_OUTPUT_FILE" required:"true"`
	PromTargetServerPort int    `envconfig:"PROM_TARGET_SERVER_PORT" required:"true"`

	// Prometheus data source.
	DsTemplateFile string `envconfig:"DS_TEMPLATE_FILE" required:"true"`
	DsOutputFile   string `envconfig:"DS_OUTPUT_FILE" required:"true"`
	DsServerPort   int    `envconfig:"DS_SERVER_PORT" required:"true"`

	// CoreDNS.
	CorednsHost string `envconfig:"COREDNS_HOST" required:"true"`
	Domain      string `envconfig:"DOMAIN" required:"true"`

	// DNS servers.
	DnsServer1Name string `envconfig:"DNS_SERVER_1_NAME" required:"true"`
	DnsServer1Port int    `envconfig:"DNS_SERVER_1_PORT" required:"true"`
	DnsServer2Name string `envconfig:"DNS_SERVER_2_NAME" required:"true"`
	DnsServer2Port int    `envconfig:"DNS_SERVER_2_PORT" required:"true"`
	DnsServer3Name string `envconfig:"DNS_SERVER_3_NAME" required:"true"`
	DnsServer3Port int    `envconfig:"DNS_SERVER_3_PORT" required:"true"`

	// Random server latency.
	RslPeriodInSeconds int `envconfig:"RSL_PERIOD_IN_SECONDS" required:"true"`
	RslMinValueInMs    int `envconfig:"RSL_MIN_VALUE_IN_MS" required:"true"`
	RslMaxValueInMs    int `envconfig:"RSL_MAX_VALUE_IN_MS" required:"true"`

	// Randomly stop/start dns servers.
	SsMinPeriodInSeconds int `envconfig:"SS_MIN_PERIOD_IN_SECONDS" required:"true"`
	SsMaxPeriodInSeconds int `envconfig:"SS_MAX_PERIOD_IN_SECONDS" required:"true"`

	// Tester.
	WaitTimeForServers int `envconfig:"WAIT_TIME_FOR_SERVERS" required:"true"`
}

// For ease of unit testing.
var (
	godotenvLoad     = godotenv.Load
	envconfigProcess = envconfig.Process
)

// Read reads the environment variables from the given file and returns a Config.
func Read() (*Config, error) {
	if err := godotenvLoad(); err != nil {
		return nil, errors.Wrap(err, "loading env vars")
	}
	config := new(Config)
	if err := envconfigProcess("", config); err != nil {
		return nil, errors.Wrap(err, "processing env vars")
	}
	return config, nil
}
