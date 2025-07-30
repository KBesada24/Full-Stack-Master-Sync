package models

import "time"

// LogEntry represents a log entry from frontend or backend
type LogEntry struct {
	ID         string                 `json:"id" validate:"required"`
	Timestamp  time.Time              `json:"timestamp" validate:"required"`
	Level      string                 `json:"level" validate:"required,oneof=error warn info debug trace"`
	Source     string                 `json:"source" validate:"required,oneof=frontend backend"`
	Message    string                 `json:"message" validate:"required,min=1"`
	Context    map[string]interface{} `json:"context"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	Component  string                 `json:"component,omitempty"`
	Function   string                 `json:"function,omitempty"`
	LineNumber int                    `json:"line_number,omitempty"`
}

// LogSubmissionRequest represents a request to submit logs
type LogSubmissionRequest struct {
	Logs     []LogEntry        `json:"logs" validate:"required"`
	BatchID  string            `json:"batch_id"`
	Source   string            `json:"source" validate:"required,oneof=frontend backend"`
	Metadata map[string]string `json:"metadata"`
}

// LogSubmissionResponse represents the response after log submission
type LogSubmissionResponse struct {
	Accepted    int       `json:"accepted"`
	Rejected    int       `json:"rejected"`
	BatchID     string    `json:"batch_id"`
	ProcessedAt time.Time `json:"processed_at"`
	Errors      []string  `json:"errors,omitempty"`
}

// LogAnalysisRequest represents a request for log analysis
type LogAnalysisRequest struct {
	TimeRange   TimeRange         `json:"time_range"`
	Levels      []string          `json:"levels"`
	Sources     []string          `json:"sources"`
	Components  []string          `json:"components"`
	SearchQuery string            `json:"search_query"`
	Filters     map[string]string `json:"filters"`
	Limit       int               `json:"limit" validate:"min=1,max=1000"`
}

// LogAnalysisResponse represents the response from log analysis
type LogAnalysisResponse struct {
	Summary     string        `json:"summary"`
	Issues      []LogIssue    `json:"issues"`
	Patterns    []LogPattern  `json:"patterns"`
	Suggestions []string      `json:"suggestions"`
	Statistics  LogStatistics `json:"statistics"`
	AnalyzedAt  time.Time     `json:"analyzed_at"`
}

// LogIssue represents an issue identified in logs
type LogIssue struct {
	Type               string     `json:"type" validate:"required,oneof=error_spike performance_degradation security_concern data_inconsistency"`
	Count              int        `json:"count" validate:"min=1"`
	FirstSeen          time.Time  `json:"first_seen"`
	LastSeen           time.Time  `json:"last_seen"`
	Description        string     `json:"description" validate:"required"`
	Severity           string     `json:"severity" validate:"required,oneof=critical high medium low"`
	Solution           string     `json:"solution"`
	AffectedComponents []string   `json:"affected_components"`
	SampleLogs         []LogEntry `json:"sample_logs,omitempty"`
}

// LogPattern represents a pattern identified in logs
type LogPattern struct {
	Pattern     string    `json:"pattern" validate:"required"`
	Frequency   int       `json:"frequency" validate:"min=1"`
	Description string    `json:"description"`
	Category    string    `json:"category" validate:"oneof=error warning info performance security"`
	Trend       string    `json:"trend" validate:"oneof=increasing decreasing stable"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

// LogStatistics represents statistical information about logs
type LogStatistics struct {
	TotalLogs     int                   `json:"total_logs"`
	LogsByLevel   map[string]int        `json:"logs_by_level"`
	LogsBySource  map[string]int        `json:"logs_by_source"`
	LogsByHour    map[string]int        `json:"logs_by_hour"`
	ErrorRate     float64               `json:"error_rate"`
	TopErrors     []LogErrorSummary     `json:"top_errors"`
	TopComponents []LogComponentSummary `json:"top_components"`
}

// LogErrorSummary represents a summary of a specific error
type LogErrorSummary struct {
	Message   string    `json:"message"`
	Count     int       `json:"count"`
	Component string    `json:"component"`
	LastSeen  time.Time `json:"last_seen"`
}

// LogComponentSummary represents a summary of logs by component
type LogComponentSummary struct {
	Component  string  `json:"component"`
	Count      int     `json:"count"`
	ErrorCount int     `json:"error_count"`
	ErrorRate  float64 `json:"error_rate"`
}

// LogAlertRequest represents a request to create a log alert
type LogAlertRequest struct {
	Name        string            `json:"name" validate:"required,min=1"`
	Description string            `json:"description"`
	Conditions  []AlertCondition  `json:"conditions" validate:"required,min=1"`
	Actions     []AlertAction     `json:"actions" validate:"required,min=1"`
	Enabled     bool              `json:"enabled"`
	Metadata    map[string]string `json:"metadata"`
}

// AlertCondition represents a condition for triggering an alert
type AlertCondition struct {
	Type       string      `json:"type" validate:"required,oneof=error_count error_rate pattern_match threshold"`
	Field      string      `json:"field"`
	Operator   string      `json:"operator" validate:"required,oneof=greater_than less_than equals contains"`
	Value      interface{} `json:"value" validate:"required"`
	TimeWindow string      `json:"time_window" validate:"required,oneof=1m 5m 15m 30m 1h 6h 12h 24h"`
}

// AlertAction represents an action to take when an alert is triggered
type AlertAction struct {
	Type   string            `json:"type" validate:"required,oneof=webhook email websocket log"`
	Config map[string]string `json:"config" validate:"required"`
}

// LogAlert represents a log alert configuration
type LogAlert struct {
	ID            string            `json:"id" validate:"required"`
	Name          string            `json:"name" validate:"required"`
	Description   string            `json:"description"`
	Conditions    []AlertCondition  `json:"conditions"`
	Actions       []AlertAction     `json:"actions"`
	Enabled       bool              `json:"enabled"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	LastTriggered *time.Time        `json:"last_triggered,omitempty"`
	TriggerCount  int               `json:"trigger_count"`
	Metadata      map[string]string `json:"metadata"`
}
