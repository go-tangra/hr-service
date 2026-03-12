package metrics

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	commonMetrics "github.com/go-tangra/go-tangra-common/metrics"
)

const namespace = "tangra"
const subsystem = "hr"

// Collector holds all Prometheus metrics for the HR module.
type Collector struct {
	log    *log.Helper
	server *commonMetrics.MetricsServer

	// Absence type metrics
	AbsenceTypesTotal prometheus.Gauge

	// Leave request metrics
	LeaveRequestsByStatus *prometheus.GaugeVec

	// Leave allowance metrics
	LeaveAllowancesTotal prometheus.Gauge

	// gRPC request metrics
	RequestDuration *prometheus.HistogramVec
	RequestsTotal   *prometheus.CounterVec
}

// NewCollector creates and registers all HR Prometheus metrics.
func NewCollector(ctx *bootstrap.Context) *Collector {
	c := &Collector{
		log: ctx.NewLoggerHelper("hr/metrics"),

		AbsenceTypesTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "absence_types_total",
			Help:      "Total number of absence types.",
		}),

		LeaveRequestsByStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "leave_requests_by_status",
			Help:      "Number of leave requests by status.",
		}, []string{"status"}),

		LeaveAllowancesTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "leave_allowances_total",
			Help:      "Total number of leave allowances.",
		}),

		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "grpc_request_duration_seconds",
			Help:      "Histogram of gRPC request durations in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method"}),

		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "grpc_requests_total",
			Help:      "Total number of gRPC requests by method and status.",
		}, []string{"method", "status"}),
	}

	prometheus.MustRegister(
		c.AbsenceTypesTotal,
		c.LeaveRequestsByStatus,
		c.LeaveAllowancesTotal,
		c.RequestDuration,
		c.RequestsTotal,
	)

	addr := os.Getenv("METRICS_ADDR")
	if addr == "" {
		addr = ":10210"
	}
	c.server = commonMetrics.NewMetricsServer(addr, nil, ctx.GetLogger())

	go func() {
		if err := c.server.Start(); err != nil {
			c.log.Errorf("Metrics server failed: %v", err)
		}
	}()

	return c
}

// Stop shuts down the metrics HTTP server.
func (c *Collector) Stop(ctx context.Context) {
	if c.server != nil {
		c.server.Stop(ctx)
	}
}

// Middleware returns a Kratos middleware that records gRPC request metrics.
func (c *Collector) Middleware() middleware.Middleware {
	return commonMetrics.NewServerMiddleware(c.RequestDuration, c.RequestsTotal)
}

// --- Absence type helpers ---

// AbsenceTypeCreated increments the absence type counter.
func (c *Collector) AbsenceTypeCreated() {
	c.AbsenceTypesTotal.Inc()
}

// AbsenceTypeDeleted decrements the absence type counter.
func (c *Collector) AbsenceTypeDeleted() {
	c.AbsenceTypesTotal.Dec()
}

// --- Leave request helpers ---

// LeaveRequestCreated increments the leave request counter for the given status.
func (c *Collector) LeaveRequestCreated(status string) {
	c.LeaveRequestsByStatus.WithLabelValues(status).Inc()
}

// LeaveRequestDeleted decrements the leave request counter for the given status.
func (c *Collector) LeaveRequestDeleted(status string) {
	c.LeaveRequestsByStatus.WithLabelValues(status).Dec()
}

// LeaveRequestStatusChanged adjusts the status gauge when a leave request's status changes.
func (c *Collector) LeaveRequestStatusChanged(oldStatus, newStatus string) {
	c.LeaveRequestsByStatus.WithLabelValues(oldStatus).Dec()
	c.LeaveRequestsByStatus.WithLabelValues(newStatus).Inc()
}

// --- Leave allowance helpers ---

// AllowanceCreated increments the leave allowance counter.
func (c *Collector) AllowanceCreated() {
	c.LeaveAllowancesTotal.Inc()
}

// AllowanceDeleted decrements the leave allowance counter.
func (c *Collector) AllowanceDeleted() {
	c.LeaveAllowancesTotal.Dec()
}
