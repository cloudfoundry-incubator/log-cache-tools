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
	metricstore_client "github.com/pivotal/metric-store/pkg/client"
	"github.com/pivotal/metric-store/pkg/rpc/metricstore_v1"
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

func Walk(client *metricstore_client.Client, metricName string, writePointsToDatadog func([]*metricstore_v1.Point) bool) {
	startTime := time.Now().Add(-time.Minute)

	for {
		time.Sleep(time.Second)
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		points, err := client.Read(
			ctx,
			metricName,
			startTime,
			metricstore_client.WithEndTime(time.Now().Add(-5*time.Second)),
		)

		if err != nil {
			log.Println(err)
			continue
		}

		if len(points) == 0 {
			continue
		}

		log.Printf("Writing %d point(s) for %s", len(points), metricName)
		writePointsToDatadog(points)
		startTime = time.Unix(0, points[len(points)-1].GetTimestamp()+1)
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
