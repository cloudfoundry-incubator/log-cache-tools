package datadog

import (
	"fmt"
	"log"
	"time"

	"github.com/cloudfoundry/metric-store-release/src/pkg/rpc/metricstore_v1"
	datadogapi "github.com/zorkian/go-datadog-api"
)

type Client interface {
	PostMetrics(m []datadogapi.Metric) error
}

func NewPointWriter(c Client, host string, staticTags []string) func(*metricstore_v1.PromQL_Series) time.Time {
	return func(series *metricstore_v1.PromQL_Series) time.Time {
		var metricName string

		datadogTags := append(make([]string, 0), staticTags...)
		labels := series.GetMetric()
		for labelName, labelValue := range labels {
			if labelName == "__name__" {
				metricName = labelValue
				continue
			}
			datadogTags = append(datadogTags, labelName+":"+labelValue)
		}

		if labels["source_id"] != "" {
			metricName = fmt.Sprintf("%s.%s", labels["source_id"], metricName)
		}

		points := series.GetPoints()
		var ps []datadogapi.DataPoint
		for _, point := range points {
			ps = append(ps, toDataPoint(point.GetTime(), point.GetValue()))
		}

		latestTime := time.Unix(0, 0)
		if len(ps) > 0 {
			metricType := "gauge"
			metrics := []datadogapi.Metric{{
				Metric: &metricName,
				Points: ps,
				Type:   &metricType,
				Host:   &host,
				Tags:   datadogTags,
			}}
			log.Printf("Writing %d point(s)", len(metrics))

			err := c.PostMetrics(metrics)
			if err != nil {
				log.Printf("failed to write metrics to DataDog: %s", err)
			}

			latestTime = MillisecondsToTime(points[len(points)-1].GetTime())
		}

		return latestTime
	}
}

func toDataPoint(timeInMilliseconds int64, value float64) datadogapi.DataPoint {
	tf := float64(MillisecondsToSeconds(timeInMilliseconds))
	return datadogapi.DataPoint{&tf, &value}
}

func MillisecondsToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}

func MillisecondsToSeconds(ms int64) int64 {
	return ms * int64(time.Second/time.Millisecond)
}
