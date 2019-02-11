package main

import (
	"log"
	"net/url"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
	"github.com/nu7hatch/gouuid"
)

type Config struct {
	SourceID       string `env:"SOURCE_ID,        required"`
	SourceHostname string `env:"SOURCE_HOST_NAME, required"`
	GroupName      string `env:"GROUP_NAME"`

	UAAURL       string `env:"UAA_URL,       required"`
	ClientID     string `env:"CLIENT_ID,     required"`
	ClientSecret string `env:"CLIENT_SECRET, noreport"`

	Username string `env:"USERNAME, required"`
	Password string `env:"PASSWORD, required, noreport"`

	LogCacheHTTPAddr string   `env:"LOG_CACHE_HTTP_ADDR, required"`
	SyslogURL        *url.URL `env:"SYSLOG_URL,          required"`

	SkipCertVerify bool `env:"SKIP_CERT_VERIFY"`

	DialTimeout time.Duration `env:"DIAL_TIMEOUT"`
	IOTimeout   time.Duration `env:"IO_TIMEOUT"`
	KeepAlive   time.Duration `env:"KEEP_ALIVE"`
}

func LoadConfig() Config {
	defaultGroup, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("unable to generate uuid: %s", err)
	}

	cfg := Config{
		SkipCertVerify: false,
		GroupName:      defaultGroup.String(),
		KeepAlive:      10 * time.Second,
		DialTimeout:    5 * time.Second,
		IOTimeout:      time.Minute,
	}
	if err := envstruct.Load(&cfg); err != nil {
		log.Fatalf("failed to load config from environment: %s", err)
	}

	return cfg
}