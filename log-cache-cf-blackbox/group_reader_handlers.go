package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/go-log-cache/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	uuid "github.com/nu7hatch/gouuid"
)

func groupLatencyTestRunner(cfg Config, httpClient *http.Client, size int) *LatencyTestResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := buildGroupClient(cfg, httpClient)

	groupUUID, err := uuid.NewV4()
	if err != nil {
		log.Printf("unable to create groupUUID: %s", err)
		return nil
	}
	groupName := groupUUID.String()

	sIDs, err := sourceIDs(httpClient, cfg, size)
	if err != nil {
		log.Printf("unable to get sourceIDs: %s", err)
		return nil
	}
	go maintainGroup(ctx, groupName, sIDs, client)

	latCtx, _ := context.WithTimeout(ctx, 10*time.Second)
	resultData, err := measureLatency(
		latCtx,
		client.BuildReader(rand.Uint64()),
		groupName,
	)
	if err != nil {
		log.Printf("error getting result data: %s", err)
		return nil
	}

	return resultData
}

func groupReliabilityTestRunner(cfg Config, httpClient *http.Client, size int) *ReliabilityTestResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := buildGroupClient(cfg, httpClient)
	reader := client.BuildReader(rand.Uint64())

	groupUUID, err := uuid.NewV4()
	if err != nil {
		log.Printf("unable to create groupUUID: %s", err)
		return nil
	}
	groupName := groupUUID.String()

	sIDs, err := sourceIDs(httpClient, cfg, size)
	if err != nil {
		log.Printf("unable to get sourceIDs: %s", err)
		return nil
	}
	go maintainGroup(ctx, groupName, sIDs, client)

	primeCtx, _ := context.WithTimeout(ctx, time.Minute)
	err = prime(primeCtx, groupName, reader)
	if err != nil {
		log.Printf("unable to prime for group: %s: %s", groupName, err)
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
	walkCtx, _ := context.WithTimeout(ctx, time.Minute)
	logcache.Walk(
		walkCtx,
		groupName,
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
		logcache.WithWalkBackoff(logcache.NewRetryBackoff(time.Second, 30)),
		logcache.WithWalkEnvelopeTypes(logcache_v1.EnvelopeType_LOG),
	)

	return &ReliabilityTestResult{
		LogsSent:        emitCount,
		LogsReceived:    receivedCount,
		BadLogsReceived: badReceivedCount,
	}
}

func buildGroupClient(cfg Config, httpClient *http.Client) *logcache.ShardGroupReaderClient {
	var localClient logcache.HTTPClient = httpClient
	if cfg.UAAAddr != "" {
		localClient = logcache.NewOauth2HTTPClient(
			cfg.UAAAddr,
			cfg.UAAClient,
			cfg.UAAClientSecret,
			logcache.WithOauth2HTTPClient(httpClient),
		)
	}
	return logcache.NewShardGroupReaderClient(
		cfg.LogCacheURL.String(),
		logcache.WithHTTPClient(localClient),
	)
}

func maintainGroup(
	ctx context.Context,
	groupName string,
	sIDs []string,
	client *logcache.ShardGroupReaderClient,
) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		for _, sID := range sIDs {
			shardGroupCtx, _ := context.WithTimeout(ctx, time.Second)
			err := client.SetShardGroup(shardGroupCtx, groupName, sID)
			if err != nil {
				log.Printf("unable to set shard group: %s", err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			continue
		}
	}
}

func sourceIDs(httpClient *http.Client, cfg Config, size int) ([]string, error) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	meta, err := client.Meta(ctx)
	if err != nil {
		return nil, err
	}

	cs := make([]count, 0)
	for k, v := range meta {
		if k == cfg.Source() {
			continue
		}
		cs = append(cs, count{
			count:    int(v.Count),
			sourceID: k,
		})
	}

	sort.Sort(counts(cs))

	sourceIDs := make([]string, 0, size)
	for _, k := range cs {
		if k.sourceID == cfg.Source() {
			continue
		}
		if len(sourceIDs) < size-1 {
			sourceIDs = append(sourceIDs, k.sourceID)
			log.Printf("Using %s with count %d", k.sourceID, k.count)
		}
	}
	sourceIDs = append(sourceIDs, cfg.Source())
	if len(sourceIDs) != size {
		return nil, fmt.Errorf("Asked for %d source IDs but only found %d", size, len(sourceIDs))
	}
	return sourceIDs, nil
}

type count struct {
	sourceID string
	count    int
}

type counts []count

func (c counts) Len() int {
	return len(c)
}

func (c counts) Swap(i, j int) {
	t := c[i]
	c[i] = c[j]
	c[j] = t
}

func (c counts) Less(i, j int) bool {
	return c[i].count < c[j].count
}
