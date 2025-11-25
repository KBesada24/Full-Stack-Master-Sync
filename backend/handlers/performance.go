package handlers

import (
	"runtime"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/middleware"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
)

// PerformanceHandler handles performance monitoring endpoints
type PerformanceHandler struct {
	logger *utils.Logger
}

// NewPerformanceHandler creates a new performance handler
func NewPerformanceHandler(logger *utils.Logger) *PerformanceHandler {
	return &PerformanceHandler{
		logger: logger,
	}
}

// GetPerformanceMetrics returns current performance metrics
func (h *PerformanceHandler) GetPerformanceMetrics(c *fiber.Ctx) error {
	metrics := middleware.GetPerformanceMetrics()

	return utils.SuccessResponse(c, "Performance metrics retrieved successfully", metrics)
}

// GetMemoryStats returns detailed memory statistics
func (h *PerformanceHandler) GetMemoryStats(c *fiber.Ctx) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	stats := fiber.Map{
		"memory": fiber.Map{
			"alloc_bytes":         memStats.Alloc,
			"alloc_mb":            float64(memStats.Alloc) / 1024 / 1024,
			"total_alloc_bytes":   memStats.TotalAlloc,
			"total_alloc_mb":      float64(memStats.TotalAlloc) / 1024 / 1024,
			"sys_bytes":           memStats.Sys,
			"sys_mb":              float64(memStats.Sys) / 1024 / 1024,
			"num_gc":              memStats.NumGC,
			"gc_cpu_fraction":     memStats.GCCPUFraction,
			"heap_alloc_bytes":    memStats.HeapAlloc,
			"heap_alloc_mb":       float64(memStats.HeapAlloc) / 1024 / 1024,
			"heap_sys_bytes":      memStats.HeapSys,
			"heap_sys_mb":         float64(memStats.HeapSys) / 1024 / 1024,
			"heap_idle_bytes":     memStats.HeapIdle,
			"heap_idle_mb":        float64(memStats.HeapIdle) / 1024 / 1024,
			"heap_inuse_bytes":    memStats.HeapInuse,
			"heap_inuse_mb":       float64(memStats.HeapInuse) / 1024 / 1024,
			"heap_released_bytes": memStats.HeapReleased,
			"heap_released_mb":    float64(memStats.HeapReleased) / 1024 / 1024,
			"heap_objects":        memStats.HeapObjects,
			"stack_inuse_bytes":   memStats.StackInuse,
			"stack_inuse_mb":      float64(memStats.StackInuse) / 1024 / 1024,
			"stack_sys_bytes":     memStats.StackSys,
			"stack_sys_mb":        float64(memStats.StackSys) / 1024 / 1024,
		},
		"runtime": fiber.Map{
			"version":       runtime.Version(),
			"num_goroutine": runtime.NumGoroutine(),
			"num_cpu":       runtime.NumCPU(),
			"goos":          runtime.GOOS,
			"goarch":        runtime.GOARCH,
		},
		"timestamp": time.Now().UTC(),
	}

	return utils.SuccessResponse(c, "Memory statistics retrieved successfully", stats)
}

// GetConnectionPoolStats returns connection pool statistics
func (h *PerformanceHandler) GetConnectionPoolStats(c *fiber.Ctx) error {
	stats := utils.GetAllPoolStats()

	return utils.SuccessResponse(c, "Connection pool statistics retrieved successfully", stats)
}

// ResetPerformanceMetrics resets all performance metrics
func (h *PerformanceHandler) ResetPerformanceMetrics(c *fiber.Ctx) error {
	middleware.ResetPerformanceMetrics()

	h.logger.WithTraceID(utils.GetTraceID(c)).WithSource("performance").Info(
		"Performance metrics reset", map[string]interface{}{
			"reset_by": c.IP(),
		})

	return utils.SuccessResponse(c, "Performance metrics reset successfully", nil)
}

// TriggerGC triggers garbage collection
func (h *PerformanceHandler) TriggerGC(c *fiber.Ctx) error {
	var beforeStats, afterStats runtime.MemStats
	runtime.ReadMemStats(&beforeStats)

	runtime.GC()

	runtime.ReadMemStats(&afterStats)

	result := fiber.Map{
		"before": fiber.Map{
			"alloc_mb":      float64(beforeStats.Alloc) / 1024 / 1024,
			"heap_alloc_mb": float64(beforeStats.HeapAlloc) / 1024 / 1024,
			"num_gc":        beforeStats.NumGC,
		},
		"after": fiber.Map{
			"alloc_mb":      float64(afterStats.Alloc) / 1024 / 1024,
			"heap_alloc_mb": float64(afterStats.HeapAlloc) / 1024 / 1024,
			"num_gc":        afterStats.NumGC,
		},
		"freed_mb":  float64(beforeStats.Alloc-afterStats.Alloc) / 1024 / 1024,
		"timestamp": time.Now().UTC(),
	}

	h.logger.WithTraceID(utils.GetTraceID(c)).WithSource("performance").Info(
		"Garbage collection triggered", map[string]interface{}{
			"freed_bytes":  beforeStats.Alloc - afterStats.Alloc,
			"triggered_by": c.IP(),
		})

	return utils.SuccessResponse(c, "Garbage collection completed", result)
}

// GetSystemInfo returns system information
func (h *PerformanceHandler) GetSystemInfo(c *fiber.Ctx) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	info := fiber.Map{
		"system": fiber.Map{
			"os":            runtime.GOOS,
			"architecture":  runtime.GOARCH,
			"go_version":    runtime.Version(),
			"num_cpu":       runtime.NumCPU(),
			"num_goroutine": runtime.NumGoroutine(),
		},
		"memory": fiber.Map{
			"alloc_mb":      float64(memStats.Alloc) / 1024 / 1024,
			"sys_mb":        float64(memStats.Sys) / 1024 / 1024,
			"heap_alloc_mb": float64(memStats.HeapAlloc) / 1024 / 1024,
			"heap_sys_mb":   float64(memStats.HeapSys) / 1024 / 1024,
			"num_gc":        memStats.NumGC,
		},
		"performance":      middleware.GetPerformanceMetrics(),
		"connection_pools": utils.GetAllPoolStats(),
		"timestamp":        time.Now().UTC(),
	}

	return utils.SuccessResponse(c, "System information retrieved successfully", info)
}

// GetEndpointMetrics returns metrics for a specific endpoint
func (h *PerformanceHandler) GetEndpointMetrics(c *fiber.Ctx) error {
	method := c.Query("method", "GET")
	path := c.Query("path")

	if path == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "MISSING_PATH",
			"Path parameter is required", nil)
	}

	metrics := middleware.GetPerformanceMetrics()
	endpointKey := method + ":" + path

	endpointMetrics, exists := metrics.EndpointMetrics[endpointKey]
	if !exists {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "ENDPOINT_NOT_FOUND",
			"No metrics found for the specified endpoint", map[string]string{
				"method": method,
				"path":   path,
			})
	}

	return utils.SuccessResponse(c, "Endpoint metrics retrieved successfully", endpointMetrics)
}

// GetTopEndpoints returns the top endpoints by various metrics
func (h *PerformanceHandler) GetTopEndpoints(c *fiber.Ctx) error {
	sortBy := c.Query("sort_by", "request_count") // request_count, average_response_time, error_rate
	limit := c.QueryInt("limit", 10)

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	metrics := middleware.GetPerformanceMetrics()

	// Convert to slice for sorting
	type endpointWithKey struct {
		Key string `json:"key"`
		*middleware.EndpointMetrics
	}

	endpoints := make([]endpointWithKey, 0, len(metrics.EndpointMetrics))
	for key, endpoint := range metrics.EndpointMetrics {
		endpoints = append(endpoints, endpointWithKey{
			Key:             key,
			EndpointMetrics: endpoint,
		})
	}

	// Sort based on criteria
	switch sortBy {
	case "average_response_time":
		// Sort by average response time (descending)
		for i := 0; i < len(endpoints)-1; i++ {
			for j := i + 1; j < len(endpoints); j++ {
				if endpoints[i].AverageResponseTime < endpoints[j].AverageResponseTime {
					endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
				}
			}
		}
	case "error_rate":
		// Sort by error rate (descending)
		for i := 0; i < len(endpoints)-1; i++ {
			for j := i + 1; j < len(endpoints); j++ {
				if endpoints[i].ErrorRate < endpoints[j].ErrorRate {
					endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
				}
			}
		}
	default: // request_count
		// Sort by request count (descending)
		for i := 0; i < len(endpoints)-1; i++ {
			for j := i + 1; j < len(endpoints); j++ {
				if endpoints[i].RequestCount < endpoints[j].RequestCount {
					endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
				}
			}
		}
	}

	// Limit results
	if len(endpoints) > limit {
		endpoints = endpoints[:limit]
	}

	result := fiber.Map{
		"sort_by":   sortBy,
		"limit":     limit,
		"total":     len(metrics.EndpointMetrics),
		"endpoints": endpoints,
	}

	return utils.SuccessResponse(c, "Top endpoints retrieved successfully", result)
}

// HealthCheck returns the health status of the performance monitoring system
func (h *PerformanceHandler) HealthCheck(c *fiber.Ctx) error {
	metrics := middleware.GetPerformanceMetrics()

	health := fiber.Map{
		"status": "healthy",
		"checks": fiber.Map{
			"metrics_collection": "ok",
			"memory_monitoring":  "ok",
			"connection_pools":   "ok",
		},
		"metrics_summary": fiber.Map{
			"total_requests":      metrics.RequestCount,
			"active_connections":  metrics.ActiveConnections,
			"average_response_ms": metrics.AverageResponseTime,
			"tracked_endpoints":   len(metrics.EndpointMetrics),
		},
		"timestamp": time.Now().UTC(),
	}

	return utils.SuccessResponse(c, "Performance monitoring system is healthy", health)
}
