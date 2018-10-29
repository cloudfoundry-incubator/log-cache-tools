package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
	logcache "code.cloudfoundry.org/log-cache/pkg/client"
	"code.cloudfoundry.org/loggregator-tools/log-cache-forwarders/pkg/egress/datadog"
	datadogapi "github.com/zorkian/go-datadog-api"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg := LoadConfig()
	envstruct.WriteReport(&cfg)

	fmt.Println("Creating client for", cfg.LogCacheHTTPAddr)
	client := logcache.NewClient(
		cfg.LogCacheHTTPAddr,
		logcache.WithHTTPClient(newOauth2HTTPClient(cfg)),
	)

	if len(cfg.SourceIDList) == 0 {
		log.Fatalf("Must provide a list of source IDs")
	}

	ddc := datadogapi.NewClient(cfg.DatadogAPIKey, "")
	visitor := datadog.Visitor(ddc, cfg.MetricHost, strings.Split(cfg.DatadogTags, ","))

	for _, sourceId := range cfg.SourceIDList {
		fmt.Println("Begrudgingly starting a walker for", sourceId)
		go logcache.Walk(
			context.Background(),
			sourceId,
			logcache.Visitor(visitor),
			client.Read,
			logcache.WithWalkStartTime(time.Now()),
			logcache.WithWalkBackoff(logcache.NewAlwaysRetryBackoff(250*time.Millisecond)),
			logcache.WithWalkLogger(log.New(os.Stderr, "", log.LstdFlags)),
		)
	}

	http.ListenAndServe(":8080", nil)
}

func newOauth2HTTPClient(cfg Config) *logcache.Oauth2HTTPClient {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipCertVerify,
			},
		},
		Timeout: 5 * time.Second,
	}

	return logcache.NewOauth2HTTPClient(
		cfg.UAAAddr,
		cfg.ClientID,
		cfg.ClientSecret,
		logcache.WithOauth2HTTPClient(client),
	)
}
