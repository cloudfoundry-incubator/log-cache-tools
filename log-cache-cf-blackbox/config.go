package main

import (
	"encoding/json"
	"net/url"
	"time"
)

type Config struct {
	LogCacheURL *url.URL `env:"LOG_CACHE_URL, required, report"`

	VCapApp VCapApp `env:"VCAP_APPLICATION, report"`

	UAAAddr         string `env:"UAA_ADDR,          required, report"`
	UAAClient       string `env:"UAA_CLIENT,        required"`
	UAAClientSecret string `env:"UAA_CLIENT_SECRET, required"`

	SkipSSLValidation bool `env:"SKIP_SSL_VALIDATION, report"`

	WalkDelay    time.Duration `env:"WALK_DELAY",    required, report"`
	TestInterval time.Duration `env:"TEST_INTERVAL", required, report"`

	DatadogAPIKey     string `env:"DATADOG_API_KEY,     required"`
	DatadogOriginHost string `env:"DATADOG_ORIGIN_HOST, required, report"`
	MetricPrefix      string `env:"METRIC_PREFIX,       required, report"`
}

func (c Config) Source() string {
	return c.VCapApp.ApplicationID
}

type VCapApp struct {
	ApplicationID string `json:"application_id, report"`
}

func (v *VCapApp) UnmarshalEnv(jsonData string) error {
	if jsonData == "" {
		return nil
	}
	return json.Unmarshal([]byte(jsonData), &v)
}
