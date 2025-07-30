package models

import (
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
)

func TestAIRequestValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		request   AIRequest
		wantValid bool
		wantError string
	}{
		{
			name: "valid JavaScript suggestion request",
			request: AIRequest{
				Code:        "function hello() { console.log('world'); }",
				Language:    "javascript",
				Context:     "This is a simple greeting function",
				RequestType: "suggestion",
				Metadata: map[string]string{
					"file": "greeting.js",
					"line": "1",
				},
			},
			wantValid: true,
		},
		{
			name: "valid TypeScript debug request",
			request: AIRequest{
				Code:        "const user: User = { name: 'John', age: '30' };",
				Language:    "typescript",
				Context:     "Type error in user object",
				RequestType: "debug",
				Metadata:    map[string]string{},
			},
			wantValid: true,
		},
		{
			name: "valid Go optimization request",
			request: AIRequest{
				Code:        "for i := 0; i < len(slice); i++ { fmt.Println(slice[i]) }",
				Language:    "go",
				Context:     "Loop optimization needed",
				RequestType: "optimize",
				Metadata:    map[string]string{},
			},
			wantValid: true,
		},
		{
			name: "missing code",
			request: AIRequest{
				Code:        "",
				Language:    "javascript",
				Context:     "Test context",
				RequestType: "suggestion",
				Metadata:    map[string]string{},
			},
			wantValid: false,
			wantError: "code",
		},
		{
			name: "invalid language",
			request: AIRequest{
				Code:        "print('hello')",
				Language:    "invalid_language",
				Context:     "Test context",
				RequestType: "suggestion",
				Metadata:    map[string]string{},
			},
			wantValid: false,
			wantError: "language",
		},
		{
			name: "invalid request type",
			request: AIRequest{
				Code:        "console.log('test');",
				Language:    "javascript",
				Context:     "Test context",
				RequestType: "invalid_type",
				Metadata:    map[string]string{},
			},
			wantValid: false,
			wantError: "request_type",
		},
		{
			name: "context too long",
			request: AIRequest{
				Code:        "console.log('test');",
				Language:    "javascript",
				Context:     string(make([]byte, 2001)), // 2001 characters
				RequestType: "suggestion",
				Metadata:    map[string]string{},
			},
			wantValid: false,
			wantError: "context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.request)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid request, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid request, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestAIResponseValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		response  AIResponse
		wantValid bool
		wantError string
	}{
		{
			name: "valid AI response",
			response: AIResponse{
				Suggestions: []Suggestion{
					{
						Type:        "improvement",
						Description: "Use const instead of let for immutable variables",
						Code:        "const message = 'Hello World';",
						LineNumber:  1,
						Priority:    "medium",
						Reasoning:   "Const prevents accidental reassignment",
					},
				},
				Analysis:    "The code looks good overall with minor improvements possible",
				Confidence:  0.85,
				RequestID:   "req-123",
				ProcessedAt: time.Now(),
			},
			wantValid: true,
		},
		{
			name: "valid response with empty suggestions",
			response: AIResponse{
				Suggestions: []Suggestion{},
				Analysis:    "No improvements needed",
				Confidence:  0.95,
				RequestID:   "req-456",
				ProcessedAt: time.Now(),
			},
			wantValid: true,
		},
		{
			name: "invalid confidence too high",
			response: AIResponse{
				Suggestions: []Suggestion{},
				Analysis:    "Test analysis",
				Confidence:  1.5,
				RequestID:   "req-789",
				ProcessedAt: time.Now(),
			},
			wantValid: false,
			wantError: "confidence",
		},
		{
			name: "invalid confidence too low",
			response: AIResponse{
				Suggestions: []Suggestion{},
				Analysis:    "Test analysis",
				Confidence:  -0.1,
				RequestID:   "req-789",
				ProcessedAt: time.Now(),
			},
			wantValid: false,
			wantError: "confidence",
		},
		{
			name: "missing request ID",
			response: AIResponse{
				Suggestions: []Suggestion{},
				Analysis:    "Test analysis",
				Confidence:  0.8,
				RequestID:   "",
				ProcessedAt: time.Now(),
			},
			wantValid: false,
			wantError: "request_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.response)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid response, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid response, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestSuggestionValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name       string
		suggestion Suggestion
		wantValid  bool
		wantError  string
	}{
		{
			name: "valid improvement suggestion",
			suggestion: Suggestion{
				Type:        "improvement",
				Description: "Use arrow function for better readability",
				Code:        "const greet = (name) => `Hello, ${name}!`;",
				LineNumber:  5,
				Priority:    "low",
				Reasoning:   "Arrow functions are more concise",
			},
			wantValid: true,
		},
		{
			name: "valid fix suggestion",
			suggestion: Suggestion{
				Type:        "fix",
				Description: "Fix undefined variable error",
				Code:        "const userName = user.name || 'Anonymous';",
				LineNumber:  10,
				Priority:    "high",
				Reasoning:   "Prevents runtime errors",
			},
			wantValid: true,
		},
		{
			name: "invalid type",
			suggestion: Suggestion{
				Type:        "invalid_type",
				Description: "Test description",
				Code:        "test code",
				LineNumber:  1,
				Priority:    "medium",
				Reasoning:   "Test reasoning",
			},
			wantValid: false,
			wantError: "type",
		},
		{
			name: "missing description",
			suggestion: Suggestion{
				Type:        "improvement",
				Description: "",
				Code:        "test code",
				LineNumber:  1,
				Priority:    "medium",
				Reasoning:   "Test reasoning",
			},
			wantValid: false,
			wantError: "description",
		},
		{
			name: "invalid priority",
			suggestion: Suggestion{
				Type:        "improvement",
				Description: "Test description",
				Code:        "test code",
				LineNumber:  1,
				Priority:    "invalid_priority",
				Reasoning:   "Test reasoning",
			},
			wantValid: false,
			wantError: "priority",
		},
		{
			name: "negative line number",
			suggestion: Suggestion{
				Type:        "improvement",
				Description: "Test description",
				Code:        "test code",
				LineNumber:  -1,
				Priority:    "medium",
				Reasoning:   "Test reasoning",
			},
			wantValid: false,
			wantError: "line_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.suggestion)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid suggestion, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid suggestion, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestTimeRangeValidation(t *testing.T) {
	validator := utils.NewValidator()

	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)

	tests := []struct {
		name      string
		timeRange TimeRange
		wantValid bool
		wantError string
	}{
		{
			name: "valid time range",
			timeRange: TimeRange{
				Start: oneHourAgo,
				End:   now,
			},
			wantValid: true,
		},
		{
			name: "missing start time",
			timeRange: TimeRange{
				Start: time.Time{},
				End:   now,
			},
			wantValid: false,
			wantError: "start",
		},
		{
			name: "missing end time",
			timeRange: TimeRange{
				Start: oneHourAgo,
				End:   time.Time{},
			},
			wantValid: false,
			wantError: "end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.timeRange)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid time range, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid time range, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}
