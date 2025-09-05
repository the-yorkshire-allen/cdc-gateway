
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	DeliveriesSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cdcgw_deliveries_sent_total", Help: "Successful deliveries",
	}, []string{"sink"})
	DeliveriesFail = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cdcgw_deliveries_fail_total", Help: "Failed deliveries",
	}, []string{"sink"})
)

func init() {
	prometheus.MustRegister(DeliveriesSent, DeliveriesFail)
}
