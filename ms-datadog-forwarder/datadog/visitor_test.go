package datadog_test

import (
	"time"

	"github.com/cloudfoundry-incubator/log-cache-tools/ms-datadog-forwarder/datadog"
	"github.com/pivotal/metric-store/pkg/rpc/metricstore_v1"
	datadogapi "github.com/zorkian/go-datadog-api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewPointWriter()", func() {
	It("writes points to the datadog client", func() {
		datadogClient := &stubDatadogClient{}
		writePoints := datadog.NewPointWriter(datadogClient, "hostname", []string{"tag-1", "tag-2"})

		latestTime := writePoints(&metricstore_v1.PromQL_Series{
			Metric: map[string]string{
				"__name__": "counter-a",
			},
			Points: []*metricstore_v1.PromQL_Point{
				{
					Time:  1000*int64(time.Millisecond),
					Value: 123,
				},
				{
					Time:  3000,
					Value: 456,
				},
			},
		})
		Expect(latestTime).To(Equal(time.Unix(3, 0)))

		latestTime = writePoints(&metricstore_v1.PromQL_Series{
			Metric: map[string]string{
				"__name__": "counter-b",
			},
			Points: []*metricstore_v1.PromQL_Point{
				{
					Time:  1000,
					Value: 456,
				},
			},
		})
		Expect(latestTime).To(Equal(time.Unix(1, 0)))

		Expect(datadogClient.metrics).To(HaveLen(2))

		m := datadogClient.metrics[0]
		Expect(*m.Type).To(Equal("gauge"))
		Expect(*m.Metric).To(Equal("counter-a"))
		Expect(*m.Host).To(Equal("hostname"))
		Expect(m.Tags).To(ConsistOf("tag-1", "tag-2"))

		Expect(m.Points).To(HaveLen(2))

		p := m.Points[0]

		Expect(*p[0]).To(Equal(float64(1)))
		Expect(*p[1]).To(Equal(float64(123)))
	})

	Context("when source_id is present on the Point", func() {
		It("write points with a metric name that includes the source id", func() {
			datadogClient := &stubDatadogClient{}
			writePoints := datadog.NewPointWriter(datadogClient, "hostname", []string{})

			writePoints(&metricstore_v1.PromQL_Series{
				Metric: map[string]string{
					"__name__":  "counter-a",
					"source_id": "counter-id-1",
				},
				Points: []*metricstore_v1.PromQL_Point{
					{
						Time:  1000,
						Value: 123,
					},
				},
			})

			m := datadogClient.metrics[0]
			Expect(*m.Metric).To(Equal("counter-id-1.counter-a"))
		})
	})

	Context("when envelopes are empty", func() {
		It("does not post metrics", func() {
			datadogClient := &stubDatadogClient{}
			writePoints := datadog.NewPointWriter(datadogClient, "hostname", []string{})

			latestTime := writePoints(&metricstore_v1.PromQL_Series{})
			Expect(datadogClient.postMetricsCalled).To(BeFalse())
			Expect(latestTime).To(Equal(time.Unix(0, 0)))

			latestTime = writePoints(nil)
			Expect(datadogClient.postMetricsCalled).To(BeFalse())
			Expect(latestTime).To(Equal(time.Unix(0, 0)))
		})
	})
})

type stubDatadogClient struct {
	postMetricsCalled bool
	metrics           []datadogapi.Metric
}

func (s *stubDatadogClient) PostMetrics(m []datadogapi.Metric) error {
	s.postMetricsCalled = true
	s.metrics = append(s.metrics, m...)
	return nil
}
