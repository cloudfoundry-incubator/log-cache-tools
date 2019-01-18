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

func NewPointWriter(c Client, host string, staticTags []string) func(points []*metricstore_v1.Point) bool {
	return func(points []*metricstore_v1.Point) bool {
		var metrics []datadog.Metric

		for _, point := range points {
			datadogTags := append(make([]string, 0), staticTags...)

			labels := point.GetLabels()
			for labelName, labelValue := range labels {
				datadogTags = append(datadogTags, labelName+":"+labelValue)
			}

			metricName := point.GetName()
			if labels["source_id"] != "" {
				metricName = fmt.Sprintf("%s.%s", labels["source_id"], metricName)
			}

			metricType := "gauge"
			metrics = append(metrics, datadog.Metric{
				Metric: &metricName,
				Points: toDataPoint(point.Timestamp, point.GetValue()),
				Type:   &metricType,
				Host:   &host,
				Tags:   datadogTags,
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
