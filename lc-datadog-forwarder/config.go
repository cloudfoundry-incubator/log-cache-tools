package main

import (
	"log"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

type Config struct {
	SourceIDList []string `env:"SOURCE_ID_LIST,report"`

	DatadogAPIKey string `env:"DATADOG_API_KEY,required"`
	MetricHost    string `env:"METRIC_HOST,required,report"`

	// DatadogTags are a comma separated list of tags to be set on each
	// metric.
	DatadogTags string `env:"DATADOG_TAGS,report"`

	UAAAddr      string `env:"UAA_ADDR,required,report"`
	ClientID     string `env:"CLIENT_ID,required,report"`
	ClientSecret string `env:"CLIENT_SECRET,required"`

	LogCacheHTTPAddr string `env:"LOG_CACHE_HTTP_ADDR,required,report"`

	SkipCertVerify bool `env:"SKIP_CERT_VERIFY,report"`
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
