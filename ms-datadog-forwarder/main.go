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
	"github.com/cloudfoundry-incubator/log-cache-tools/ms-datadog-forwarder/datadog"
	"github.com/pivotal/metric-store/pkg/client"
	datadogapi "github.com/zorkian/go-datadog-api"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg := LoadConfig()
	envstruct.WriteReport(&cfg)

	fmt.Println("Creating client for", cfg.DataSourceHTTPAddr)
	client := metricstore_client.NewClient(
		cfg.DataSourceHTTPAddr,
		metricstore_client.WithHTTPClient(newOauth2HTTPClient(cfg)),
	)

	if len(cfg.ForwardedMetrics) == 0 {
		log.Fatalf("Must provide a set of forwarded metrics")
	}

	ddc := datadogapi.NewClient(cfg.DatadogAPIKey, "")
	visitor := datadog.Visitor(ddc, cfg.MetricHost, strings.Split(cfg.DatadogTags, ","))

	for sourceId, associatedMetrics := range cfg.ForwardedMetrics {
		for _, metricName := range associatedMetrics {
			fmt.Println("Begrudgingly starting a walker for", sourceId, metricName)
			go metricstore_client.Walk(
				context.Background(),
				metricName,
				sourceId,
				metricstore_client.Visitor(visitor),
				client.Read,
				metricstore_client.WithWalkStartTime(time.Now()),
				metricstore_client.WithWalkBackoff(metricstore_client.NewAlwaysRetryBackoff(250*time.Millisecond)),
				metricstore_client.WithWalkLogger(log.New(os.Stderr, "", log.LstdFlags)),
			)
		}
	}

	http.ListenAndServe(":8080", nil)
}

func newOauth2HTTPClient(cfg Config) *metricstore_client.Oauth2HTTPClient {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipCertVerify,
			},
		},
		Timeout: 5 * time.Second,
	}

	return metricstore_client.NewOauth2HTTPClient(
		cfg.UAAAddr,
		cfg.ClientID,
		cfg.ClientSecret,
		metricstore_client.WithOauth2HTTPClient(client),
	)
}
