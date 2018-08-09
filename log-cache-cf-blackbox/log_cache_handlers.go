package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
)

func latencyTestRunner(cfg Config, httpClient *http.Client) *LatencyTestResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var localClient logcache.HTTPClient = httpClient
	if cfg.UAAAddr != "" {
		localClient = logcache.NewOauth2HTTPClient(
			cfg.UAAAddr,
			cfg.UAAClient,
			cfg.UAAClientSecret,
			logcache.WithOauth2HTTPClient(httpClient),
		)
	}
	client := logcache.NewClient(
		cfg.LogCacheURL.String(),
		logcache.WithHTTPClient(localClient),
	)

	latCtx, _ := context.WithTimeout(ctx, 10*time.Second)
	testResult, err := measureLatency(latCtx, client.Read, cfg.Source())

	if err != nil {
		log.Printf("error getting result data: %s", err)
		return nil
	}

	return testResult
}

func reliabilityTestRunner(cfg Config, httpClient *http.Client) *ReliabilityTestResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := buildClient(cfg, httpClient)
	reader := client.Read

	primeCtx, _ := context.WithTimeout(ctx, time.Minute)
	err := prime(primeCtx, cfg.Source(), reader)
	if err != nil {
		log.Printf("unable to prime for source id: %s: %s", cfg.Source(), err)
		return nil
	}

	emitCount := 10000
	prefix := fmt.Sprintf("%d - ", time.Now().UnixNano())

	start := time.Now()
	for i := 0; i < emitCount; i++ {
		log.Printf("%s%d", prefix, i)
		time.Sleep(time.Millisecond)
	}
	end := time.Now()

	var (
		receivedCount    int
		badReceivedCount int
	)
	walkCtx, _ := context.WithTimeout(ctx, 40*time.Second)
	logcache.Walk(
		walkCtx,
		cfg.Source(),
		func(envelopes []*loggregator_v2.Envelope) bool {
			for _, e := range envelopes {
				if strings.Contains(string(e.GetLog().GetPayload()), prefix) {
					receivedCount++
				} else {
					badReceivedCount++
				}
			}
			return receivedCount < emitCount
		},
		reader,
		logcache.WithWalkStartTime(start),
		logcache.WithWalkEndTime(end),
		logcache.WithWalkBackoff(logcache.NewAlwaysRetryBackoff(time.Second)),
		logcache.WithWalkDelay(cfg.WalkDelay),
	)

	return &ReliabilityTestResult{
		LogsSent:        emitCount,
		LogsReceived:    receivedCount,
		BadLogsReceived: badReceivedCount,
	}
}

func buildClient(cfg Config, httpClient *http.Client) *logcache.Client {
	var localClient logcache.HTTPClient = httpClient
	if cfg.UAAAddr != "" {
		localClient = logcache.NewOauth2HTTPClient(
			cfg.UAAAddr,
			cfg.UAAClient,
			cfg.UAAClientSecret,
			logcache.WithOauth2HTTPClient(httpClient),
		)
	}
	return logcache.NewClient(
		cfg.LogCacheURL.String(),
		logcache.WithHTTPClient(localClient),
	)
}
