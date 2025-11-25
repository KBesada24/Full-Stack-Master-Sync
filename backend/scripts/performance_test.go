package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// PerformanceTestConfig holds configuration for performance testing
type PerformanceTestConfig struct {
	BaseURL         string
	ConcurrentUsers int
	RequestsPerUser int
	TestDuration    time.Duration
}

// TestResult holds the results of a performance test
type TestResult struct {
	TotalRequests  int           `json:"total_requests"`
	SuccessfulReqs int           `json:"successful_requests"`
	FailedReqs     int           `json:"failed_requests"`
	AverageLatency time.Duration `json:"average_latency"`
	MinLatency     time.Duration `json:"min_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	RequestsPerSec float64       `json:"requests_per_second"`
	TestDuration   time.Duration `json:"test_duration"`
	ErrorRate      float64       `json:"error_rate"`
}

func main() {
	config := PerformanceTestConfig{
		BaseURL:         "http://localhost:8080",
		ConcurrentUsers: 10,
		RequestsPerUser: 100,
		TestDuration:    30 * time.Second,
	}

	fmt.Println("🚀 Starting Performance Tests...")
	fmt.Printf("Base URL: %s\n", config.BaseURL)
	fmt.Printf("Concurrent Users: %d\n", config.ConcurrentUsers)
	fmt.Printf("Requests per User: %d\n", config.RequestsPerUser)
	fmt.Printf("Test Duration: %v\n", config.TestDuration)
	fmt.Println()

	// Test different endpoints
	endpoints := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"Health Check", "GET", "/health", nil},
		{"Performance Metrics", "GET", "/api/performance/metrics", nil},
		{"Memory Stats", "GET", "/api/performance/memory", nil},
		{"System Info", "GET", "/api/performance/system", nil},
		{"Sync Status", "GET", "/api/sync/status", nil},
		{"AI Status", "GET", "/api/ai/status", nil},
		{"Testing Status", "GET", "/api/testing/status", nil},
		{"Logging Status", "GET", "/api/logs/status", nil},
	}

	for _, endpoint := range endpoints {
		fmt.Printf("Testing %s (%s %s)...\n", endpoint.name, endpoint.method, endpoint.path)
		result := runPerformanceTest(config, endpoint.method, endpoint.path, endpoint.body)
		printTestResult(endpoint.name, result)
		fmt.Println()

		// Wait a bit between tests
		time.Sleep(2 * time.Second)
	}

	// Test with POST requests
	fmt.Println("Testing POST endpoints...")

	// Test sync connect
	syncBody := map[string]interface{}{
		"frontend_url": "http://localhost:3000",
		"backend_url":  "http://localhost:8080",
		"environment":  "test",
	}

	fmt.Printf("Testing Sync Connect (POST /api/sync/connect)...\n")
	result := runPerformanceTest(config, "POST", "/api/sync/connect", syncBody)
	printTestResult("Sync Connect", result)
	fmt.Println()

	fmt.Println("✅ Performance tests completed!")
}

func runPerformanceTest(config PerformanceTestConfig, method, path string, body interface{}) TestResult {
	var wg sync.WaitGroup
	var mu sync.Mutex

	results := make([]time.Duration, 0, config.ConcurrentUsers*config.RequestsPerUser)
	errors := 0

	startTime := time.Now()

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Run concurrent users
	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < config.RequestsPerUser; j++ {
				reqStart := time.Now()

				// Create request
				var req *http.Request
				var err error

				if body != nil {
					bodyBytes, _ := json.Marshal(body)
					req, err = http.NewRequest(method, config.BaseURL+path, bytes.NewReader(bodyBytes))
					if err == nil {
						req.Header.Set("Content-Type", "application/json")
					}
				} else {
					req, err = http.NewRequest(method, config.BaseURL+path, nil)
				}

				if err != nil {
					mu.Lock()
					errors++
					mu.Unlock()
					continue
				}

				// Make request
				resp, err := client.Do(req)
				latency := time.Since(reqStart)

				mu.Lock()
				if err != nil || resp.StatusCode >= 400 {
					errors++
				} else {
					results = append(results, latency)
				}
				mu.Unlock()

				if resp != nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	testDuration := time.Since(startTime)

	// Calculate statistics
	totalRequests := config.ConcurrentUsers * config.RequestsPerUser
	successfulReqs := len(results)
	failedReqs := errors

	var avgLatency, minLatency, maxLatency time.Duration
	if len(results) > 0 {
		var totalLatency time.Duration
		minLatency = results[0]
		maxLatency = results[0]

		for _, latency := range results {
			totalLatency += latency
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}

		avgLatency = totalLatency / time.Duration(len(results))
	}

	requestsPerSec := float64(successfulReqs) / testDuration.Seconds()
	errorRate := float64(failedReqs) / float64(totalRequests) * 100

	return TestResult{
		TotalRequests:  totalRequests,
		SuccessfulReqs: successfulReqs,
		FailedReqs:     failedReqs,
		AverageLatency: avgLatency,
		MinLatency:     minLatency,
		MaxLatency:     maxLatency,
		RequestsPerSec: requestsPerSec,
		TestDuration:   testDuration,
		ErrorRate:      errorRate,
	}
}

func printTestResult(testName string, result TestResult) {
	fmt.Printf("📊 %s Results:\n", testName)
	fmt.Printf("  Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("  Successful: %d\n", result.SuccessfulReqs)
	fmt.Printf("  Failed: %d\n", result.FailedReqs)
	fmt.Printf("  Error Rate: %.2f%%\n", result.ErrorRate)
	fmt.Printf("  Average Latency: %v\n", result.AverageLatency)
	fmt.Printf("  Min Latency: %v\n", result.MinLatency)
	fmt.Printf("  Max Latency: %v\n", result.MaxLatency)
	fmt.Printf("  Requests/sec: %.2f\n", result.RequestsPerSec)
	fmt.Printf("  Test Duration: %v\n", result.TestDuration)

	// Performance assessment
	if result.ErrorRate == 0 && result.RequestsPerSec > 100 {
		fmt.Printf("  ✅ Excellent performance!\n")
	} else if result.ErrorRate < 5 && result.RequestsPerSec > 50 {
		fmt.Printf("  ✅ Good performance\n")
	} else if result.ErrorRate < 10 {
		fmt.Printf("  ⚠️  Acceptable performance\n")
	} else {
		fmt.Printf("  ❌ Poor performance - needs optimization\n")
	}
}
