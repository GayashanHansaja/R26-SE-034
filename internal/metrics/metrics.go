package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ERP Requests
	ERPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "erp_requests_total",
		Help: "Total number of outbound ERP requests",
	}, []string{"method", "path", "status"})

	ERPLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "erp_request_duration_seconds",
		Help:    "Latency of outbound ERP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// MCP Tool Invocations
	ToolInvocationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mcp_tool_invocations_total",
		Help: "Total number of MCP tool calls",
	}, []string{"tool", "cache_status"})

	ToolLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mcp_tool_duration_seconds",
		Help:    "Latency of MCP tool calls",
		Buckets: prometheus.DefBuckets,
	}, []string{"tool"})

	// Cache Stats
	CacheHitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Total number of cache hits",
	}, []string{"type"}) // exact, semantic

	CacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Total number of cache misses",
	})
)
