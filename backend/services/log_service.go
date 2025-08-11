package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/google/uuid"
)

// AIServiceInterface defines the interface for AI service integration
type AIServiceInterface interface {
	IsAvailable() bool
	AnalyzeLogs(ctx context.Context, req *models.AILogAnalysisRequest) (*models.AILogAnalysisResponse, error)
}

// LogService handles log storage, analysis, and alerting
type LogService struct {
	logs      []models.LogEntry
	alerts    []models.LogAlert
	aiService AIServiceInterface
	wsHub     WebSocketBroadcaster
	mu        sync.RWMutex
	logger    *utils.Logger
}

// NewLogService creates a new log service instance
func NewLogService(aiService AIServiceInterface, wsHub WebSocketBroadcaster) *LogService {
	return &LogService{
		logs:      make([]models.LogEntry, 0),
		alerts:    make([]models.LogAlert, 0),
		aiService: aiService,
		wsHub:     wsHub,
		logger:    utils.GetLogger(),
	}
}

// SubmitLogs processes and stores log entries from frontend or backend
func (s *LogService) SubmitLogs(ctx context.Context, req *models.LogSubmissionRequest) (*models.LogSubmissionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accepted := 0
	rejected := 0
	errors := make([]string, 0)
	batchID := req.BatchID
	if batchID == "" {
		batchID = uuid.New().String()
	}

	s.logger.Info("Processing log submission", map[string]interface{}{
		"batch_id":  batchID,
		"source":    req.Source,
		"log_count": len(req.Logs),
		"metadata":  req.Metadata,
	})

	// Process each log entry
	for i, logEntry := range req.Logs {
		// Validate log entry
		if err := s.validateLogEntry(&logEntry); err != nil {
			rejected++
			errors = append(errors, fmt.Sprintf("Log %d: %v", i+1, err))
			continue
		}

		// Set default values if missing
		if logEntry.ID == "" {
			logEntry.ID = uuid.New().String()
		}
		if logEntry.Timestamp.IsZero() {
			logEntry.Timestamp = time.Now()
		}

		// Store the log entry
		s.logs = append(s.logs, logEntry)
		accepted++

		// Check for critical log events and send WebSocket notifications
		if s.isCriticalLogEvent(&logEntry) {
			s.sendCriticalLogAlert(&logEntry)
		}
	}

	// Trim logs if we have too many (keep last 10000)
	if len(s.logs) > 10000 {
		s.logs = s.logs[len(s.logs)-10000:]
	}

	response := &models.LogSubmissionResponse{
		Accepted:    accepted,
		Rejected:    rejected,
		BatchID:     batchID,
		ProcessedAt: time.Now(),
		Errors:      errors,
	}

	s.logger.Info("Log submission processed", map[string]interface{}{
		"batch_id": batchID,
		"accepted": accepted,
		"rejected": rejected,
	})

	return response, nil
}

// AnalyzeLogs performs analysis on stored logs with optional AI integration
func (s *LogService) AnalyzeLogs(ctx context.Context, req *models.LogAnalysisRequest) (*models.LogAnalysisResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.Info("Starting log analysis", map[string]interface{}{
		"time_range":   req.TimeRange,
		"levels":       req.Levels,
		"sources":      req.Sources,
		"components":   req.Components,
		"search_query": req.SearchQuery,
		"limit":        req.Limit,
	})

	// Filter logs based on request criteria
	filteredLogs := s.filterLogs(req)

	// Apply limit
	if req.Limit > 0 && len(filteredLogs) > req.Limit {
		filteredLogs = filteredLogs[:req.Limit]
	}

	// Perform basic analysis
	issues := s.detectIssues(filteredLogs)
	patterns := s.detectPatterns(filteredLogs)
	statistics := s.calculateStatistics(filteredLogs)

	// Generate summary
	summary := s.generateSummary(filteredLogs, issues, patterns)

	// Generate basic suggestions
	suggestions := s.generateSuggestions(issues, patterns)

	// Try AI-enhanced analysis if available
	if s.aiService != nil && s.aiService.IsAvailable() && len(filteredLogs) > 0 {
		aiAnalysis, err := s.performAIAnalysis(ctx, filteredLogs)
		if err != nil {
			s.logger.Warn("AI analysis failed, using basic analysis", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			// Enhance results with AI analysis
			summary = aiAnalysis.Summary
			if len(aiAnalysis.Issues) > 0 {
				issues = append(issues, aiAnalysis.Issues...)
			}
			if len(aiAnalysis.Patterns) > 0 {
				patterns = append(patterns, aiAnalysis.Patterns...)
			}
			if len(aiAnalysis.Suggestions) > 0 {
				suggestions = append(suggestions, aiAnalysis.Suggestions...)
			}
		}
	}

	response := &models.LogAnalysisResponse{
		Summary:     summary,
		Issues:      issues,
		Patterns:    patterns,
		Suggestions: suggestions,
		Statistics:  statistics,
		AnalyzedAt:  time.Now(),
	}

	s.logger.Info("Log analysis completed", map[string]interface{}{
		"analyzed_logs": len(filteredLogs),
		"issues_found":  len(issues),
		"patterns":      len(patterns),
		"suggestions":   len(suggestions),
	})

	return response, nil
}

// validateLogEntry validates a log entry
func (s *LogService) validateLogEntry(entry *models.LogEntry) error {
	if entry.Message == "" {
		return fmt.Errorf("message is required")
	}
	if entry.Level == "" {
		return fmt.Errorf("level is required")
	}
	if entry.Source == "" {
		return fmt.Errorf("source is required")
	}

	// Validate level
	validLevels := map[string]bool{
		"error": true, "warn": true, "info": true, "debug": true, "trace": true,
	}
	if !validLevels[entry.Level] {
		return fmt.Errorf("invalid level: %s", entry.Level)
	}

	// Validate source
	validSources := map[string]bool{
		"frontend": true, "backend": true,
	}
	if !validSources[entry.Source] {
		return fmt.Errorf("invalid source: %s", entry.Source)
	}

	return nil
}

// filterLogs filters logs based on analysis request criteria
func (s *LogService) filterLogs(req *models.LogAnalysisRequest) []models.LogEntry {
	filtered := make([]models.LogEntry, 0)

	for _, log := range s.logs {
		// Time range filter
		if !req.TimeRange.Start.IsZero() && log.Timestamp.Before(req.TimeRange.Start) {
			continue
		}
		if !req.TimeRange.End.IsZero() && log.Timestamp.After(req.TimeRange.End) {
			continue
		}

		// Level filter
		if len(req.Levels) > 0 {
			found := false
			for _, level := range req.Levels {
				if log.Level == level {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Source filter
		if len(req.Sources) > 0 {
			found := false
			for _, source := range req.Sources {
				if log.Source == source {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Component filter
		if len(req.Components) > 0 {
			found := false
			for _, component := range req.Components {
				if log.Component == component {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Search query filter
		if req.SearchQuery != "" {
			query := strings.ToLower(req.SearchQuery)
			if !strings.Contains(strings.ToLower(log.Message), query) &&
				!strings.Contains(strings.ToLower(log.Component), query) &&
				!strings.Contains(strings.ToLower(log.Function), query) {
				continue
			}
		}

		// Custom filters
		if len(req.Filters) > 0 {
			match := true
			for key, value := range req.Filters {
				switch key {
				case "user_id":
					if log.UserID != value {
						match = false
					}
				case "session_id":
					if log.SessionID != value {
						match = false
					}
				default:
					// Check in context
					if contextValue, exists := log.Context[key]; exists {
						if fmt.Sprintf("%v", contextValue) != value {
							match = false
						}
					} else {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}

		filtered = append(filtered, log)
	}

	// Sort by timestamp (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	return filtered
}

// detectIssues identifies issues in the filtered logs
func (s *LogService) detectIssues(logs []models.LogEntry) []models.LogIssue {
	issues := make([]models.LogIssue, 0)
	errorCounts := make(map[string]int)
	errorTimes := make(map[string][]time.Time)

	// Count errors and track timing
	for _, log := range logs {
		if log.Level == "error" {
			key := log.Message
			if log.Component != "" {
				key = fmt.Sprintf("%s:%s", log.Component, log.Message)
			}
			errorCounts[key]++
			errorTimes[key] = append(errorTimes[key], log.Timestamp)
		}
	}

	// Detect error spikes
	for errorMsg, count := range errorCounts {
		if count >= 5 { // Threshold for error spike
			times := errorTimes[errorMsg]
			sort.Slice(times, func(i, j int) bool {
				return times[i].Before(times[j])
			})

			issue := models.LogIssue{
				Type:        "error_spike",
				Count:       count,
				FirstSeen:   times[0],
				LastSeen:    times[len(times)-1],
				Description: fmt.Sprintf("High frequency of error: %s", errorMsg),
				Severity:    s.determineSeverity(count),
				Solution:    "Investigate the root cause of this recurring error",
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// detectPatterns identifies patterns in the filtered logs
func (s *LogService) detectPatterns(logs []models.LogEntry) []models.LogPattern {
	patterns := make([]models.LogPattern, 0)
	messageCounts := make(map[string]int)
	messageTimes := make(map[string][]time.Time)

	// Count message patterns
	for _, log := range logs {
		// Extract pattern from message (simplified)
		pattern := s.extractPattern(log.Message)
		messageCounts[pattern]++
		messageTimes[pattern] = append(messageTimes[pattern], log.Timestamp)
	}

	// Create patterns for frequent messages
	for pattern, count := range messageCounts {
		if count >= 3 { // Threshold for pattern detection
			times := messageTimes[pattern]
			sort.Slice(times, func(i, j int) bool {
				return times[i].Before(times[j])
			})

			logPattern := models.LogPattern{
				Pattern:     pattern,
				Frequency:   count,
				Description: fmt.Sprintf("Recurring pattern detected: %s", pattern),
				Category:    "info",
				Trend:       "stable",
				FirstSeen:   times[0],
				LastSeen:    times[len(times)-1],
			}
			patterns = append(patterns, logPattern)
		}
	}

	return patterns
}

// calculateStatistics calculates statistical information about logs
func (s *LogService) calculateStatistics(logs []models.LogEntry) models.LogStatistics {
	stats := models.LogStatistics{
		TotalLogs:     len(logs),
		LogsByLevel:   make(map[string]int),
		LogsBySource:  make(map[string]int),
		LogsByHour:    make(map[string]int),
		TopErrors:     make([]models.LogErrorSummary, 0),
		TopComponents: make([]models.LogComponentSummary, 0),
	}

	errorCounts := make(map[string]int)
	componentCounts := make(map[string]int)
	componentErrors := make(map[string]int)

	for _, log := range logs {
		// Count by level
		stats.LogsByLevel[log.Level]++

		// Count by source
		stats.LogsBySource[log.Source]++

		// Count by hour
		hour := log.Timestamp.Format("2006-01-02 15:00")
		stats.LogsByHour[hour]++

		// Count errors
		if log.Level == "error" {
			errorCounts[log.Message]++
		}

		// Count by component
		if log.Component != "" {
			componentCounts[log.Component]++
			if log.Level == "error" {
				componentErrors[log.Component]++
			}
		}
	}

	// Calculate error rate
	if stats.TotalLogs > 0 {
		errorCount := stats.LogsByLevel["error"]
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalLogs) * 100
	}

	// Create top errors list
	type errorCount struct {
		message string
		count   int
	}
	errorList := make([]errorCount, 0)
	for msg, count := range errorCounts {
		errorList = append(errorList, errorCount{msg, count})
	}
	sort.Slice(errorList, func(i, j int) bool {
		return errorList[i].count > errorList[j].count
	})

	for i, err := range errorList {
		if i >= 10 { // Top 10 errors
			break
		}
		stats.TopErrors = append(stats.TopErrors, models.LogErrorSummary{
			Message:   err.message,
			Count:     err.count,
			Component: "unknown",
			LastSeen:  time.Now(), // Simplified
		})
	}

	// Create top components list
	for component, count := range componentCounts {
		errorCount := componentErrors[component]
		errorRate := float64(0)
		if count > 0 {
			errorRate = float64(errorCount) / float64(count) * 100
		}

		stats.TopComponents = append(stats.TopComponents, models.LogComponentSummary{
			Component:  component,
			Count:      count,
			ErrorCount: errorCount,
			ErrorRate:  errorRate,
		})
	}

	// Sort components by count
	sort.Slice(stats.TopComponents, func(i, j int) bool {
		return stats.TopComponents[i].Count > stats.TopComponents[j].Count
	})

	// Keep top 10 components
	if len(stats.TopComponents) > 10 {
		stats.TopComponents = stats.TopComponents[:10]
	}

	return stats
}

// generateSummary creates a summary of the log analysis
func (s *LogService) generateSummary(logs []models.LogEntry, issues []models.LogIssue, patterns []models.LogPattern) string {
	if len(logs) == 0 {
		return "No logs found matching the specified criteria."
	}

	summary := fmt.Sprintf("Analyzed %d log entries. ", len(logs))

	errorCount := 0
	warnCount := 0
	for _, log := range logs {
		if log.Level == "error" {
			errorCount++
		} else if log.Level == "warn" {
			warnCount++
		}
	}

	if errorCount > 0 {
		summary += fmt.Sprintf("Found %d errors ", errorCount)
	}
	if warnCount > 0 {
		summary += fmt.Sprintf("and %d warnings. ", warnCount)
	}

	if len(issues) > 0 {
		summary += fmt.Sprintf("Identified %d issues requiring attention. ", len(issues))
	}

	if len(patterns) > 0 {
		summary += fmt.Sprintf("Detected %d recurring patterns.", len(patterns))
	}

	return summary
}

// generateSuggestions creates suggestions based on analysis results
func (s *LogService) generateSuggestions(issues []models.LogIssue, patterns []models.LogPattern) []string {
	suggestions := make([]string, 0)

	if len(issues) > 0 {
		suggestions = append(suggestions, "Review and address the identified issues to improve system stability")
	}

	if len(patterns) > 0 {
		suggestions = append(suggestions, "Monitor recurring patterns to identify potential optimizations")
	}

	// Add general suggestions
	suggestions = append(suggestions,
		"Implement proper error handling and logging practices",
		"Set up automated alerts for critical errors",
		"Regular log analysis can help prevent issues before they impact users",
	)

	return suggestions
}

// performAIAnalysis performs AI-enhanced log analysis
func (s *LogService) performAIAnalysis(ctx context.Context, logs []models.LogEntry) (*models.AILogAnalysisResponse, error) {
	// Limit logs for AI analysis to avoid token limits
	analysisLogs := logs
	if len(logs) > 20 {
		analysisLogs = logs[:20]
	}

	aiReq := &models.AILogAnalysisRequest{
		Logs:         analysisLogs,
		AnalysisType: "error_detection",
	}

	return s.aiService.AnalyzeLogs(ctx, aiReq)
}

// isCriticalLogEvent determines if a log event is critical
func (s *LogService) isCriticalLogEvent(log *models.LogEntry) bool {
	// Critical conditions
	if log.Level == "error" {
		return true
	}

	// Check for critical keywords in message
	criticalKeywords := []string{
		"panic", "fatal", "crash", "security", "breach", "unauthorized",
		"database connection", "out of memory", "disk full",
	}

	message := strings.ToLower(log.Message)
	for _, keyword := range criticalKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}

	return false
}

// sendCriticalLogAlert sends a WebSocket notification for critical log events
func (s *LogService) sendCriticalLogAlert(log *models.LogEntry) {
	if s.wsHub == nil {
		return
	}

	alert := map[string]interface{}{
		"type":        "critical_log_event",
		"log_id":      log.ID,
		"level":       log.Level,
		"source":      log.Source,
		"message":     log.Message,
		"component":   log.Component,
		"timestamp":   log.Timestamp,
		"stack_trace": log.StackTrace,
	}

	s.wsHub.BroadcastToAll("log_alert", alert)

	s.logger.Warn("Critical log event detected and broadcasted", map[string]interface{}{
		"log_id":    log.ID,
		"level":     log.Level,
		"source":    log.Source,
		"component": log.Component,
	})
}

// extractPattern extracts a pattern from a log message (simplified)
func (s *LogService) extractPattern(message string) string {
	// Simple pattern extraction - replace numbers and IDs with placeholders
	pattern := message

	// Replace UUIDs
	pattern = strings.ReplaceAll(pattern, uuid.New().String(), "[UUID]")

	// Replace numbers (simplified)
	words := strings.Fields(pattern)
	for i, word := range words {
		if len(word) > 0 && (word[0] >= '0' && word[0] <= '9') {
			words[i] = "[NUMBER]"
		}
	}

	return strings.Join(words, " ")
}

// determineSeverity determines the severity based on error count
func (s *LogService) determineSeverity(count int) string {
	if count >= 20 {
		return "critical"
	} else if count >= 10 {
		return "high"
	} else if count >= 5 {
		return "medium"
	}
	return "low"
}

// GetLogCount returns the total number of stored logs
func (s *LogService) GetLogCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.logs)
}

// ClearLogs clears all stored logs (for testing or maintenance)
func (s *LogService) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = make([]models.LogEntry, 0)
	s.logger.Info("All logs cleared", nil)
}
