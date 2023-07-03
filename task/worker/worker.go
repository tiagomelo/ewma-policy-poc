package worker

import (
	"context"
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tiagomelo/ewma-policy-poc/digger"
	"github.com/tiagomelo/ewma-policy-poc/screen/stats"
)

var (
	failedDnsRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "failed_dns_requests_total",
			Help: "Number of failed DNS requests.",
		},
		[]string{"domain"},
	)
)

func init() {
	prometheus.MustRegister(failedDnsRequests)
}

type Worker struct {
	Domain string
	Digger *digger.Digger
	Logger *log.Logger
	Stats  *stats.Statistics
}

func (w *Worker) Work(ctx context.Context) {
	if err := w.Digger.Dig(w.Domain); err != nil {
		w.Logger.Printf(`error when digging domain "%s": %v`, w.Domain, err)
		w.Stats.IncrTotalFailedDnsRequests()
		failedDnsRequests.With(prometheus.Labels{"domain": w.Domain}).Inc()
	}
}
