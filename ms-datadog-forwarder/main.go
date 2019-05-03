package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
	"github.com/cloudfoundry-incubator/log-cache-tools/ms-datadog-forwarder/datadog"
	metricstore_client "github.com/cloudfoundry/metric-store-release/src/pkg/client"
	"github.com/cloudfoundry/metric-store-release/src/pkg/rpc/metricstore_v1"
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

	datadogClient := datadogapi.NewClient(cfg.DatadogAPIKey, "")
	writePoints := datadog.NewPointWriter(datadogClient, cfg.MetricHost, strings.Split(cfg.DatadogTags, ","))

	for _, associatedMetrics := range cfg.ForwardedMetrics {
		for _, metricName := range associatedMetrics {
			go Walk(client, metricName, writePoints)
		}
	}

	http.ListenAndServe(":8080", nil)
}

func Walk(client *metricstore_client.Client, query string, writePointsToDatadog func(*metricstore_v1.PromQL_Series) time.Time) {
	startTime := time.Now().Add(-time.Minute)

	for {
		time.Sleep(time.Second)
		ctx, _ := context.WithTimeout(context.Background(), time.Second)

		log.Printf("Querying for blackbox metrics with: %s", query)
		res, err := client.PromQLRange(
			ctx,
			query,
			metricstore_client.WithPromQLStart(startTime),
			metricstore_client.WithPromQLEnd(time.Now().Add(-5*time.Second)),
			metricstore_client.WithPromQLStep("1s"),
		)

		if err != nil {
			log.Println(err)
			continue
		}

		var latestTime time.Time
		for _, series := range res.GetMatrix().GetSeries() {
			latestTime = writePointsToDatadog(series)

			if startTime.Before(latestTime) {
				startTime = latestTime.Add(time.Millisecond)
			}
		}
	}
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
