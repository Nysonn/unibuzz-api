package config

import (
	"github.com/prometheus/client_golang/prometheus"
)

var HTTPRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "endpoint"},
)

var HTTPDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of HTTP requests",
	},
	[]string{"endpoint"},
)

func InitMetrics() {

	prometheus.MustRegister(HTTPRequests)
	prometheus.MustRegister(HTTPDuration)
}
