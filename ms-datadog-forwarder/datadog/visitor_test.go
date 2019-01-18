package datadog_test

import (
	"github.com/cloudfoundry-incubator/log-cache-tools/ms-datadog-forwarder/datadog"
	"github.com/pivotal/metric-store/pkg/rpc/metricstore_v1"
	datadogapi "github.com/zorkian/go-datadog-api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewPointWriter", func() {
	It("writes points to the datadog client", func() {
		datadogClient := &stubDatadogClient{}
		writePoints := datadog.NewPointWriter(datadogClient, "hostname", []string{"tag-1", "tag-2"})

		cont := writePoints([]*metricstore_v1.Point{
			{
				Timestamp: 1000000000,
				Name:      "counter-a",
				Value:     123,
			},
			{
				Timestamp: 1000000000,
				Name:      "counter-b",
				Value:     456,
			},
		})

		Expect(cont).To(BeTrue())
		Expect(datadogClient.metrics).To(HaveLen(2))

		m := datadogClient.metrics[0]
		Expect(*m.Type).To(Equal("gauge"))
		Expect(*m.Metric).To(Equal("counter-a"))
		Expect(*m.Host).To(Equal("hostname"))
		Expect(m.Tags).To(ConsistOf("tag-1", "tag-2"))

		Expect(m.Points).To(HaveLen(1))

		p := m.Points[0]

		Expect(*p[0]).To(Equal(float64(1)))
		Expect(*p[1]).To(Equal(float64(123)))
	})

	Context("when source_id is present on the Point", func() {
		It("write points with a metric name that includes the source id", func() {
			datadogClient := &stubDatadogClient{}
			writePoints := datadog.NewPointWriter(datadogClient, "hostname", []string{})

			writePoints([]*metricstore_v1.Point{
				{
					Timestamp: 1000000000,
					Name:      "counter-a",
					Value:     123,
					Labels:    map[string]string{"source_id": "counter-id-1"},
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

			writePoints(nil)

			Expect(datadogClient.postMetricsCalled).To(BeFalse())
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
