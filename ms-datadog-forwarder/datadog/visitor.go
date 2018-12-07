package datadog

import (
	"fmt"
	"log"
	"time"

	"github.com/pivotal/metric-store/pkg/rpc/metricstore_v1"
	datadog "github.com/zorkian/go-datadog-api"
)

type Client interface {
	PostMetrics(m []datadog.Metric) error
}

func Visitor(c Client, host string, tags []string) func(points []*metricstore_v1.Point) bool {
	return func(points []*metricstore_v1.Point) bool {
		var metrics []datadog.Metric

		for _, point := range points {
			ddtags := append(make([]string, 0), tags...)

			tags := point.GetTags()
			for key, value := range tags {
				ddtags = append(ddtags, key+":"+value)
			}

			name := point.GetName()
			if tags["source_id"] != "" {
				name = fmt.Sprintf("%s.%s", tags["source_id"], name)
			}

			mType := "gauge"
			metrics = append(metrics, datadog.Metric{
				Metric: &name,
				Points: toDataPoint(point.Timestamp, point.GetValue()),
				Type:   &mType,
				Host:   &host,
				Tags:   ddtags,
			})
		}

		if len(metrics) > 0 {
			err := c.PostMetrics(metrics)
			if err != nil {
				log.Printf("failed to write metrics to DataDog: %s", err)
			}
		}

		return true
	}
}

func toDataPoint(x int64, y float64) []datadog.DataPoint {
	t := time.Unix(0, x)
	tf := float64(t.Unix())
	return []datadog.DataPoint{
		[2]*float64{&tf, &y},
	}
}
