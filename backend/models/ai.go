package models

import "time"

// AIRequest represents a request for AI assistance
type AIRequest struct {
	Code        string            `json:"code" validate:"required,min=1"`
	Language    string            `json:"language" validate:"required,oneof=javascript typescript python go java rust php swift kotlin dart"`
	Context     string            `json:"context" validate:"max=2000"`
	RequestType string            `json:"request_type" validate:"required,oneof=suggestion debug optimize refactor explain"`
	Metadata    map[string]string `json:"metadata"`
}

// AIResponse represents the response from AI assistance
type AIResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
	Analysis    string       `json:"analysis"`
	Confidence  float64      `json:"confidence" validate:"min=0,max=1"`
	RequestID   string       `json:"request_id" validate:"required"`
	ProcessedAt time.Time    `json:"processed_at"`
}

// Suggestion represents an AI-generated code suggestion
type Suggestion struct {
	Type        string `json:"type" validate:"required,oneof=improvement fix optimization refactor"`
	Description string `json:"description" validate:"required,min=1"`
	Code        string `json:"code"`
	LineNumber  int    `json:"line_number" validate:"min=0"`
	Priority    string `json:"priority" validate:"required,oneof=high medium low"`
	Reasoning   string `json:"reasoning"`
}

// AILogAnalysisRequest represents a request for AI log analysis
type AILogAnalysisRequest struct {
	Logs         []LogEntry        `json:"logs" validate:"required,min=1"`
	TimeRange    TimeRange         `json:"time_range"`
	Filters      map[string]string `json:"filters"`
	AnalysisType string            `json:"analysis_type" validate:"required,oneof=error_detection pattern_analysis performance_issues security_scan"`
}

// AILogAnalysisResponse represents the response from AI log analysis
type AILogAnalysisResponse struct {
	Summary     string       `json:"summary"`
	Issues      []LogIssue   `json:"issues"`
	Patterns    []LogPattern `json:"patterns"`
	Suggestions []string     `json:"suggestions"`
	AnalyzedAt  time.Time    `json:"analyzed_at"`
	Confidence  float64      `json:"confidence" validate:"min=0,max=1"`
}

// TimeRange represents a time range for filtering
type TimeRange struct {
	Start time.Time `json:"start" validate:"required"`
	End   time.Time `json:"end" validate:"required"`
}
