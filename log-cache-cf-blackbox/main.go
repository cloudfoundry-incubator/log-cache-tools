package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
	datadog "github.com/zorkian/go-datadog-api"
)

func main() {
	cfg := Config{
		WalkDelay:    time.Second,
		TestInterval: 10 * time.Minute,
	}
	err := envstruct.Load(&cfg)
	if err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

	err = envstruct.WriteReport(&cfg)
	if err != nil {
		log.Fatalf("failed to write config report: %s", err)
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	ddc := datadog.NewClient(cfg.DatadogAPIKey, "")

	t := time.NewTicker(cfg.TestInterval)

	go func() {
		for range t.C {
			log.Println("Running test loop...")
			currentTime := time.Now().UnixNano()

			latencyResult := latencyTestRunner(cfg, httpClient)
			latencyMetrics := []datadog.Metric{
				{
					Metric: metricName(cfg, "latency"),
					Points: toDataPoint(currentTime, latencyResult.Latency),
					Host:   &cfg.DatadogOriginHost,
				},
				{
					Metric: metricName(cfg, "average_query_time"),
					Points: toDataPoint(currentTime, latencyResult.AverageQueryTime),
					Host:   &cfg.DatadogOriginHost,
				},
			}

			if err := ddc.PostMetrics(latencyMetrics); err != nil {
				log.Printf("failed to write latency metrics to DataDog: %s", err)
			} else {
				log.Printf("posted %d latency metrics", len(latencyMetrics))
			}

			reliabilityResult := reliabilityTestRunner(cfg, httpClient)
			reliabilityMetrics := []datadog.Metric{
				{
					Metric: metricName(cfg, "logs_sent"),
					Points: toDataPoint(currentTime, float64(reliabilityResult.LogsSent)),
					Host:   &cfg.DatadogOriginHost,
				},
				{
					Metric: metricName(cfg, "logs_received"),
					Points: toDataPoint(currentTime, float64(reliabilityResult.LogsReceived)),
					Host:   &cfg.DatadogOriginHost,
				},
			}

			if err := ddc.PostMetrics(reliabilityMetrics); err != nil {
				log.Printf("failed to write reliability metrics to DataDog: %s", err)
			} else {
				log.Printf("posted %d reliability metrics", len(reliabilityMetrics))
			}

			// groupLatencyTestRunner(cfg, httpClient, 10)
			// groupReliabilityTestRunner(cfg, httpClient, 10)
		}
	}()

	http.ListenAndServe(":8080", nil)
}

func metricName(cfg Config, name string) *string {
	s := fmt.Sprintf("%s.%s", cfg.MetricPrefix, name)
	return &s
}

func toDataPoint(x int64, y float64) []datadog.DataPoint {
	t := time.Unix(0, x)
	tf := float64(t.Unix())
	return []datadog.DataPoint{
		[2]*float64{&tf, &y},
	}
}
