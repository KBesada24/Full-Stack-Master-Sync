package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/time/rate"
)

// AIService handles OpenAI integration with rate limiting and error handling
type AIService struct {
	client         *openai.Client
	config         *config.Config
	rateLimiter    *rate.Limiter
	mu             sync.RWMutex
	isAvailable    bool
	lastError      error
	lastCheck      time.Time
	wsHub          WebSocketBroadcaster
	circuitBreaker *utils.CircuitBreaker
	retryExecutor  *utils.RetryExecutor
	logger         *utils.Logger
}

// NewAIService creates a new AI service instance
func NewAIService(cfg *config.Config, wsHub WebSocketBroadcaster, logger *utils.Logger) *AIService {
	var client *openai.Client
	isAvailable := false

	if cfg.OpenAIAPIKey != "" {
		client = openai.NewClient(cfg.OpenAIAPIKey)
		isAvailable = true
	}

	// Rate limiter: 60 requests per minute (1 per second with burst of 10)
	limiter := rate.NewLimiter(rate.Every(time.Second), 10)

	// Circuit breaker configuration for OpenAI API
	cbConfig := &utils.CircuitBreakerConfig{
		MaxFailures:      3,
		Timeout:          60 * time.Second,
		MaxRequests:      2,
		SuccessThreshold: 2,
		Name:             "openai_api",
	}

	// Retry configuration for OpenAI API
	retryConfig := &utils.RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      500 * time.Millisecond,
		MaxDelay:          10 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
		RetryCondition: func(err error) bool {
			// Retry on rate limit and temporary errors
			errStr := strings.ToLower(err.Error())
			return strings.Contains(errStr, "rate limit") ||
				strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "temporary") ||
				strings.Contains(errStr, "service unavailable")
		},
	}

	if logger == nil {
		logger = utils.GetLogger()
	}

	return &AIService{
		client:         client,
		config:         cfg,
		rateLimiter:    limiter,
		isAvailable:    isAvailable,
		lastCheck:      time.Now(),
		wsHub:          wsHub,
		circuitBreaker: utils.NewCircuitBreaker(cbConfig, logger),
		retryExecutor:  utils.NewRetryExecutor(retryConfig, logger),
		logger:         logger,
	}
}

// IsAvailable checks if the AI service is available
func (s *AIService) IsAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isAvailable && s.client != nil
}

// GetCodeSuggestions generates code suggestions using OpenAI
func (s *AIService) GetCodeSuggestions(ctx context.Context, req *models.AIRequest) (*models.AIResponse, error) {
	if !s.IsAvailable() {
		return s.getFallbackResponse(req, "AI service is currently unavailable")
	}

	requestID := uuid.New().String()

	// Execute with circuit breaker and retry logic
	var response *models.AIResponse
	err := s.retryExecutor.Execute(ctx, func(ctx context.Context) error {
		return s.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
			// Apply rate limiting
			if err := s.rateLimiter.Wait(ctx); err != nil {
				return fmt.Errorf("rate limit exceeded: %w", err)
			}

			// Build the prompt based on request type
			prompt := s.buildCodePrompt(req)

			// Make OpenAI API call
			resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: "You are an expert code assistant. Provide helpful, accurate code suggestions and improvements.",
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					},
				},
				MaxTokens:   1000,
				Temperature: 0.3,
				TopP:        1.0,
			})

			if err != nil {
				s.updateAvailability(false, err)
				return fmt.Errorf("OpenAI API error: %w", err)
			}

			s.updateAvailability(true, nil)

			// Parse the response
			if len(resp.Choices) == 0 {
				return fmt.Errorf("no suggestions generated")
			}

			suggestions := s.parseCodeSuggestions(resp.Choices[0].Message.Content, req)

			response = &models.AIResponse{
				Suggestions: suggestions,
				Analysis:    resp.Choices[0].Message.Content,
				Confidence:  0.8, // Default confidence for OpenAI responses
				RequestID:   requestID,
				ProcessedAt: time.Now(),
			}

			return nil
		})
	})

	if err != nil {
		s.logger.WithSource("ai_service").Error("Failed to get code suggestions", err, map[string]interface{}{
			"request_id":   requestID,
			"request_type": req.RequestType,
		})
		return s.getFallbackResponse(req, fmt.Sprintf("Failed to get suggestions: %v", err))
	}

	// Broadcast AI suggestion ready notification
	s.broadcastAISuggestionReady(requestID, req.RequestType, len(response.Suggestions))

	return response, nil
}

// AnalyzeLogs analyzes logs using OpenAI
func (s *AIService) AnalyzeLogs(ctx context.Context, req *models.AILogAnalysisRequest) (*models.AILogAnalysisResponse, error) {
	if !s.IsAvailable() {
		return s.getFallbackLogAnalysis(req, "AI service is currently unavailable")
	}

	// Execute with circuit breaker and retry logic
	var response *models.AILogAnalysisResponse
	err := s.retryExecutor.Execute(ctx, func(ctx context.Context) error {
		return s.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
			// Apply rate limiting
			if err := s.rateLimiter.Wait(ctx); err != nil {
				return fmt.Errorf("rate limit exceeded: %w", err)
			}

			// Build the log analysis prompt
			prompt := s.buildLogAnalysisPrompt(req)

			// Make OpenAI API call
			resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: "You are an expert log analyst. Analyze logs to identify issues, patterns, and provide actionable suggestions.",
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					},
				},
				MaxTokens:   1500,
				Temperature: 0.2,
				TopP:        1.0,
			})

			if err != nil {
				s.updateAvailability(false, err)
				return fmt.Errorf("OpenAI API error: %w", err)
			}

			s.updateAvailability(true, nil)

			// Parse the response
			if len(resp.Choices) == 0 {
				return fmt.Errorf("no analysis generated")
			}

			analysis := s.parseLogAnalysis(resp.Choices[0].Message.Content, req)

			response = &models.AILogAnalysisResponse{
				Summary:     analysis.Summary,
				Issues:      analysis.Issues,
				Patterns:    analysis.Patterns,
				Suggestions: analysis.Suggestions,
				AnalyzedAt:  time.Now(),
				Confidence:  0.8,
			}

			return nil
		})
	})

	if err != nil {
		s.logger.WithSource("ai_service").Error("Failed to analyze logs", err, map[string]interface{}{
			"log_count":     len(req.Logs),
			"analysis_type": req.AnalysisType,
		})
		return s.getFallbackLogAnalysis(req, fmt.Sprintf("Failed to analyze logs: %v", err))
	}

	// Broadcast AI log analysis ready notification
	s.broadcastAILogAnalysisReady(len(req.Logs), len(response.Issues), len(response.Patterns))

	return response, nil
}

// buildCodePrompt creates a prompt for code suggestions
func (s *AIService) buildCodePrompt(req *models.AIRequest) string {
	var prompt strings.Builder

	switch req.RequestType {
	case "suggestion":
		prompt.WriteString("Please analyze the following code and provide suggestions for improvement:\n\n")
	case "debug":
		prompt.WriteString("Please help debug the following code and identify potential issues:\n\n")
	case "optimize":
		prompt.WriteString("Please analyze the following code and suggest optimizations:\n\n")
	case "refactor":
		prompt.WriteString("Please suggest refactoring improvements for the following code:\n\n")
	case "explain":
		prompt.WriteString("Please explain what the following code does:\n\n")
	default:
		prompt.WriteString("Please analyze the following code:\n\n")
	}

	prompt.WriteString(fmt.Sprintf("Language: %s\n", req.Language))
	prompt.WriteString(fmt.Sprintf("Code:\n```%s\n%s\n```\n\n", req.Language, req.Code))

	if req.Context != "" {
		prompt.WriteString(fmt.Sprintf("Context: %s\n\n", req.Context))
	}

	prompt.WriteString("Please provide your response in a structured format with specific suggestions, explanations, and priority levels.")

	return prompt.String()
}

// buildLogAnalysisPrompt creates a prompt for log analysis
func (s *AIService) buildLogAnalysisPrompt(req *models.AILogAnalysisRequest) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("Please analyze the following logs for %s:\n\n", req.AnalysisType))

	// Add up to 10 log entries to avoid token limits
	maxLogs := 10
	if len(req.Logs) < maxLogs {
		maxLogs = len(req.Logs)
	}

	for i := 0; i < maxLogs; i++ {
		log := req.Logs[i]
		prompt.WriteString(fmt.Sprintf("Log %d:\n", i+1))
		prompt.WriteString(fmt.Sprintf("  Timestamp: %s\n", log.Timestamp.Format(time.RFC3339)))
		prompt.WriteString(fmt.Sprintf("  Level: %s\n", log.Level))
		prompt.WriteString(fmt.Sprintf("  Source: %s\n", log.Source))
		prompt.WriteString(fmt.Sprintf("  Message: %s\n", log.Message))
		if log.StackTrace != "" {
			prompt.WriteString(fmt.Sprintf("  Stack Trace: %s\n", log.StackTrace))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("Please provide:\n")
	prompt.WriteString("1. A summary of the main issues found\n")
	prompt.WriteString("2. Specific issues with severity levels\n")
	prompt.WriteString("3. Patterns identified in the logs\n")
	prompt.WriteString("4. Actionable suggestions for resolution\n")

	return prompt.String()
}

// parseCodeSuggestions parses OpenAI response into structured suggestions
func (s *AIService) parseCodeSuggestions(content string, req *models.AIRequest) []models.Suggestion {
	// This is a simplified parser - in production, you might want more sophisticated parsing
	suggestions := []models.Suggestion{
		{
			Type:        "improvement",
			Description: "AI-generated code suggestion",
			Code:        content,
			LineNumber:  1,
			Priority:    "medium",
			Reasoning:   "Generated by OpenAI based on code analysis",
		},
	}

	return suggestions
}

// parseLogAnalysis parses OpenAI response into structured log analysis
func (s *AIService) parseLogAnalysis(content string, req *models.AILogAnalysisRequest) *models.AILogAnalysisResponse {
	// This is a simplified parser - in production, you might want more sophisticated parsing
	return &models.AILogAnalysisResponse{
		Summary: content,
		Issues: []models.LogIssue{
			{
				Type:        "general",
				Count:       1,
				FirstSeen:   time.Now(),
				LastSeen:    time.Now(),
				Description: "AI-analyzed log issue",
				Severity:    "medium",
				Solution:    "Review the AI analysis for detailed recommendations",
			},
		},
		Patterns: []models.LogPattern{
			{
				Pattern:     "AI-detected pattern",
				Frequency:   1,
				Description: "Pattern identified through AI analysis",
			},
		},
		Suggestions: []string{"Review the detailed AI analysis above"},
	}
}

// getFallbackResponse returns a fallback response when AI service is unavailable
func (s *AIService) getFallbackResponse(req *models.AIRequest, reason string) (*models.AIResponse, error) {
	requestID := uuid.New().String()

	var fallbackSuggestion models.Suggestion
	switch req.RequestType {
	case "suggestion":
		fallbackSuggestion = models.Suggestion{
			Type:        "improvement",
			Description: "AI service unavailable - consider code review best practices",
			Code:        "// AI service is currently unavailable\n// Please review your code manually",
			LineNumber:  1,
			Priority:    "low",
			Reasoning:   reason,
		}
	case "debug":
		fallbackSuggestion = models.Suggestion{
			Type:        "fix",
			Description: "AI service unavailable - use debugging tools",
			Code:        "// AI service is currently unavailable\n// Use console.log, debugger, or logging for debugging",
			LineNumber:  1,
			Priority:    "low",
			Reasoning:   reason,
		}
	default:
		fallbackSuggestion = models.Suggestion{
			Type:        "improvement",
			Description: "AI service unavailable - manual review recommended",
			Code:        "// AI service is currently unavailable\n// Please review your code manually",
			LineNumber:  1,
			Priority:    "low",
			Reasoning:   reason,
		}
	}

	response := &models.AIResponse{
		Suggestions: []models.Suggestion{fallbackSuggestion},
		Analysis:    fmt.Sprintf("AI service is currently unavailable: %s", reason),
		Confidence:  0.1, // Low confidence for fallback responses
		RequestID:   requestID,
		ProcessedAt: time.Now(),
	}

	// Broadcast AI suggestion ready notification even for fallback responses
	s.broadcastAISuggestionReady(requestID, req.RequestType, len(response.Suggestions))

	return response, nil
}

// getFallbackLogAnalysis returns a fallback log analysis when AI service is unavailable
func (s *AIService) getFallbackLogAnalysis(req *models.AILogAnalysisRequest, reason string) (*models.AILogAnalysisResponse, error) {
	return &models.AILogAnalysisResponse{
		Summary: fmt.Sprintf("AI service is currently unavailable: %s. Manual log review recommended.", reason),
		Issues: []models.LogIssue{
			{
				Type:        "service_unavailable",
				Count:       1,
				FirstSeen:   time.Now(),
				LastSeen:    time.Now(),
				Description: "AI log analysis service is unavailable",
				Severity:    "info",
				Solution:    "Review logs manually or try again later",
			},
		},
		Patterns:    []models.LogPattern{},
		Suggestions: []string{"Review logs manually", "Check AI service configuration", "Try again later"},
		AnalyzedAt:  time.Now(),
		Confidence:  0.1,
	}, nil
}

// updateAvailability updates the service availability status
func (s *AIService) updateAvailability(available bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isAvailable = available
	s.lastError = err
	s.lastCheck = time.Now()

	if err != nil {
		log.Printf("AI service availability updated: %v, error: %v", available, err)
	}
}

// GetStatus returns the current status of the AI service
func (s *AIService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"available":  s.isAvailable,
		"last_check": s.lastCheck,
	}

	if s.lastError != nil {
		status["last_error"] = s.lastError.Error()
	}

	return status
}

// HealthCheck performs a health check on the AI service
func (s *AIService) HealthCheck(ctx context.Context) error {
	if !s.IsAvailable() {
		return fmt.Errorf("AI service is not available: API key not configured")
	}

	// Execute health check with circuit breaker
	return s.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		// Apply rate limiting for health check
		if err := s.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded during health check: %w", err)
		}

		// Simple test request to verify API connectivity
		_, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Hello",
				},
			},
			MaxTokens: 5,
		})

		if err != nil {
			s.updateAvailability(false, err)
			return fmt.Errorf("AI service health check failed: %w", err)
		}

		s.updateAvailability(true, nil)
		return nil
	})
}

// broadcastAISuggestionReady broadcasts AI suggestion ready notification
func (s *AIService) broadcastAISuggestionReady(requestID, requestType string, suggestionCount int) {
	if s.wsHub == nil {
		return
	}

	notificationData := map[string]interface{}{
		"type":             "code_suggestions",
		"request_id":       requestID,
		"request_type":     requestType,
		"suggestion_count": suggestionCount,
		"timestamp":        time.Now(),
		"status":           "ready",
	}

	s.wsHub.BroadcastToAll("ai_suggestion_ready", notificationData)
}

// broadcastAILogAnalysisReady broadcasts AI log analysis ready notification
func (s *AIService) broadcastAILogAnalysisReady(logCount, issueCount, patternCount int) {
	if s.wsHub == nil {
		return
	}

	notificationData := map[string]interface{}{
		"type":           "log_analysis",
		"logs_analyzed":  logCount,
		"issues_found":   issueCount,
		"patterns_found": patternCount,
		"timestamp":      time.Now(),
		"status":         "ready",
	}

	s.wsHub.BroadcastToAll("ai_suggestion_ready", notificationData)
}
