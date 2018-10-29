package main

import (
	"encoding/json"
	"log"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

type forwardedMetrics map[string][]string

func (f *forwardedMetrics) UnmarshalEnv(data string) error {
	return json.Unmarshal([]byte(data), f)
}

type Config struct {
	ForwardedMetrics forwardedMetrics `env:"FORWARDED_METRICS, report"`

	DatadogAPIKey string `env:"DATADOG_API_KEY, required"`
	MetricHost    string `env:"METRIC_HOST, required, report"`

	// DatadogTags are a comma separated list of tags to be set on each
	// metric.
	DatadogTags string `env:"DATADOG_TAGS, report"`

	UAAAddr      string `env:"UAA_ADDR,        required, report"`
	ClientID     string `env:"CLIENT_ID,       required, report"`
	ClientSecret string `env:"CLIENT_SECRET,   required"`

	DataSourceHTTPAddr string `env:"DATA_SOURCE_HTTP_ADDR,  required, report"`

	SkipCertVerify bool `env:"SKIP_CERT_VERIFY, report"`
}

func LoadConfig() Config {
	cfg := Config{
		SkipCertVerify: false,
	}
	if err := envstruct.Load(&cfg); err != nil {
		log.Fatalf("failed to load config from environment: %s", err)
	}

	return cfg
}
