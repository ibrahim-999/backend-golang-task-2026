package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry *prometheus.Registry

	HTTPRequests *prometheus.CounterVec
	HTTPDuration *prometheus.HistogramVec
	HTTPInFlight prometheus.Gauge

	OrdersProcessed         *prometheus.CounterVec
	OrderProcessingDuration prometheus.Histogram
	InventoryReservations   *prometheus.CounterVec
	PaymentAttempts         *prometheus.CounterVec
	NotificationsDispatched *prometheus.CounterVec
	QueueDepth              *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		registry: reg,
		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "http", Name: "requests_total",
			Help: "Total HTTP requests partitioned by method, path and status.",
		}, []string{"method", "path", "status"}),
		HTTPDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "http", Name: "request_duration_seconds",
			Help: "HTTP request latency in seconds.", Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		HTTPInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Subsystem: "http", Name: "in_flight_requests",
			Help: "Number of in-flight HTTP requests.",
		}),
		OrdersProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "orders", Name: "processed_total",
			Help: "Orders processed partitioned by terminal status.",
		}, []string{"status"}),
		OrderProcessingDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "orders", Name: "processing_duration_seconds",
			Help: "Order pipeline processing latency in seconds.", Buckets: prometheus.DefBuckets,
		}),
		InventoryReservations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "inventory", Name: "reservations_total",
			Help: "Inventory reservation attempts partitioned by result.",
		}, []string{"result"}),
		PaymentAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "payment", Name: "attempts_total",
			Help: "Payment attempts partitioned by result.",
		}, []string{"result"}),
		NotificationsDispatched: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "notification", Name: "dispatched_total",
			Help: "Notifications dispatched partitioned by status.",
		}, []string{"status"}),
		QueueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Subsystem: "worker", Name: "queue_depth",
			Help: "Current depth of worker queues.",
		}, []string{"queue"}),
	}
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.HTTPRequests, m.HTTPDuration, m.HTTPInFlight,
		m.OrdersProcessed, m.OrderProcessingDuration, m.InventoryReservations,
		m.PaymentAttempts, m.NotificationsDispatched, m.QueueDepth,
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
