package utils

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	IsValid bool                       `json:"is_valid"`
	Errors  map[string]ValidationError `json:"errors,omitempty"`
}

// Validator represents a field validator
type Validator struct {
	errors map[string]ValidationError
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make(map[string]ValidationError),
	}
}

// ValidateStruct validates a struct using reflection and validation tags
func (v *Validator) ValidateStruct(s interface{}) *ValidationResult {
	v.errors = make(map[string]ValidationError)

	val := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)

	// Handle pointer to struct
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		v.addError("_root", "Value must be a struct", "")
		return v.getResult()
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get JSON tag name or use field name
		jsonTag := fieldType.Tag.Get("json")
		fieldName := fieldType.Name
		if jsonTag != "" && jsonTag != "-" {
			// Remove omitempty and other options
			if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
				fieldName = jsonTag[:commaIdx]
			} else {
				fieldName = jsonTag
			}
		}

		// Get validation tag
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// Validate field
		v.validateField(fieldName, field.Interface(), validateTag)
	}

	return v.getResult()
}

// validateField validates a single field based on validation rules
func (v *Validator) validateField(fieldName string, value interface{}, rules string) {
	ruleList := strings.Split(rules, ",")

	for _, rule := range ruleList {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		// Parse rule and parameters
		parts := strings.Split(rule, "=")
		ruleName := parts[0]
		var ruleParam string
		if len(parts) > 1 {
			ruleParam = parts[1]
		}

		// Apply validation rule
		if !v.applyRule(fieldName, value, ruleName, ruleParam) {
			break // Stop on first error for this field
		}
	}
}

// applyRule applies a specific validation rule
func (v *Validator) applyRule(fieldName string, value interface{}, ruleName, param string) bool {
	switch ruleName {
	case "required":
		return v.validateRequired(fieldName, value)
	case "email":
		return v.validateEmail(fieldName, value)
	case "url":
		return v.validateURL(fieldName, value)
	case "min":
		return v.validateMin(fieldName, value, param)
	case "max":
		return v.validateMax(fieldName, value, param)
	case "len":
		return v.validateLength(fieldName, value, param)
	case "numeric":
		return v.validateNumeric(fieldName, value)
	case "alpha":
		return v.validateAlpha(fieldName, value)
	case "alphanum":
		return v.validateAlphaNumeric(fieldName, value)
	case "oneof":
		return v.validateOneOf(fieldName, value, param)
	case "dive":
		return v.validateDive(fieldName, value, param)
	default:
		// Unknown rule, skip
		return true
	}
}

// validateRequired validates that a field is not empty
func (v *Validator) validateRequired(fieldName string, value interface{}) bool {
	if value == nil {
		v.addError(fieldName, "Field is required", "")
		return false
	}

	switch val := value.(type) {
	case string:
		if strings.TrimSpace(val) == "" {
			v.addError(fieldName, "Field is required", val)
			return false
		}
	case []interface{}:
		if len(val) == 0 {
			v.addError(fieldName, "Field is required", "")
			return false
		}
	case []string:
		if len(val) == 0 {
			v.addError(fieldName, "Field is required", "")
			return false
		}
	case map[string]interface{}:
		if len(val) == 0 {
			v.addError(fieldName, "Field is required", "")
			return false
		}
	case time.Time:
		if val.IsZero() {
			v.addError(fieldName, "Field is required", "")
			return false
		}
	}

	// Use reflection to check for zero values of other types
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice && rv.Len() == 0 {
		v.addError(fieldName, "Field is required", "")
		return false
	}

	return true
}

// validateEmail validates email format
func (v *Validator) validateEmail(fieldName string, value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		v.addError(fieldName, "Field must be a string", fmt.Sprintf("%v", value))
		return false
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		v.addError(fieldName, "Field must be a valid email address", str)
		return false
	}

	return true
}

// validateURL validates URL format
func (v *Validator) validateURL(fieldName string, value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		v.addError(fieldName, "Field must be a string", fmt.Sprintf("%v", value))
		return false
	}

	_, err := url.ParseRequestURI(str)
	if err != nil {
		v.addError(fieldName, "Field must be a valid URL", str)
		return false
	}

	return true
}

// validateMin validates minimum value/length
func (v *Validator) validateMin(fieldName string, value interface{}, param string) bool {
	minVal, err := strconv.Atoi(param)
	if err != nil {
		return true // Invalid parameter, skip validation
	}

	switch val := value.(type) {
	case string:
		if len(val) < minVal {
			v.addError(fieldName, fmt.Sprintf("Field must be at least %d characters long", minVal), val)
			return false
		}
	case int, int32, int64:
		intVal := reflect.ValueOf(val).Int()
		if intVal < int64(minVal) {
			v.addError(fieldName, fmt.Sprintf("Field must be at least %d", minVal), fmt.Sprintf("%d", intVal))
			return false
		}
	case float32, float64:
		floatVal := reflect.ValueOf(val).Float()
		if floatVal < float64(minVal) {
			v.addError(fieldName, fmt.Sprintf("Field must be at least %d", minVal), fmt.Sprintf("%.2f", floatVal))
			return false
		}
	}

	return true
}

// validateMax validates maximum value/length
func (v *Validator) validateMax(fieldName string, value interface{}, param string) bool {
	maxVal, err := strconv.Atoi(param)
	if err != nil {
		return true // Invalid parameter, skip validation
	}

	switch val := value.(type) {
	case string:
		if len(val) > maxVal {
			v.addError(fieldName, fmt.Sprintf("Field must be at most %d characters long", maxVal), val)
			return false
		}
	case int, int32, int64:
		intVal := reflect.ValueOf(val).Int()
		if intVal > int64(maxVal) {
			v.addError(fieldName, fmt.Sprintf("Field must be at most %d", maxVal), fmt.Sprintf("%d", intVal))
			return false
		}
	case float32, float64:
		floatVal := reflect.ValueOf(val).Float()
		if floatVal > float64(maxVal) {
			v.addError(fieldName, fmt.Sprintf("Field must be at most %d", maxVal), fmt.Sprintf("%.2f", floatVal))
			return false
		}
	}

	return true
}

// validateLength validates exact length
func (v *Validator) validateLength(fieldName string, value interface{}, param string) bool {
	expectedLen, err := strconv.Atoi(param)
	if err != nil {
		return true // Invalid parameter, skip validation
	}

	str, ok := value.(string)
	if !ok {
		v.addError(fieldName, "Field must be a string", fmt.Sprintf("%v", value))
		return false
	}

	if len(str) != expectedLen {
		v.addError(fieldName, fmt.Sprintf("Field must be exactly %d characters long", expectedLen), str)
		return false
	}

	return true
}

// validateNumeric validates that field contains only numbers
func (v *Validator) validateNumeric(fieldName string, value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		v.addError(fieldName, "Field must be a string", fmt.Sprintf("%v", value))
		return false
	}

	numericRegex := regexp.MustCompile(`^[0-9]+$`)
	if !numericRegex.MatchString(str) {
		v.addError(fieldName, "Field must contain only numbers", str)
		return false
	}

	return true
}

// validateAlpha validates that field contains only letters
func (v *Validator) validateAlpha(fieldName string, value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		v.addError(fieldName, "Field must be a string", fmt.Sprintf("%v", value))
		return false
	}

	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	if !alphaRegex.MatchString(str) {
		v.addError(fieldName, "Field must contain only letters", str)
		return false
	}

	return true
}

// validateAlphaNumeric validates that field contains only letters and numbers
func (v *Validator) validateAlphaNumeric(fieldName string, value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		v.addError(fieldName, "Field must be a string", fmt.Sprintf("%v", value))
		return false
	}

	alphaNumRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphaNumRegex.MatchString(str) {
		v.addError(fieldName, "Field must contain only letters and numbers", str)
		return false
	}

	return true
}

// validateOneOf validates that field value is one of the specified options
func (v *Validator) validateOneOf(fieldName string, value interface{}, param string) bool {
	str, ok := value.(string)
	if !ok {
		v.addError(fieldName, "Field must be a string", fmt.Sprintf("%v", value))
		return false
	}

	options := strings.Split(param, " ")
	for _, option := range options {
		if str == option {
			return true
		}
	}

	v.addError(fieldName, fmt.Sprintf("Field must be one of: %s", strings.Join(options, ", ")), str)
	return false
}

// validateDive validates each element in a slice
func (v *Validator) validateDive(fieldName string, value interface{}, param string) bool {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		v.addError(fieldName, "Field must be a slice for dive validation", fmt.Sprintf("%v", value))
		return false
	}

	for i := 0; i < rv.Len(); i++ {
		element := rv.Index(i).Interface()
		elementFieldName := fmt.Sprintf("%s[%d]", fieldName, i)

		// Apply the parameter validation rule to each element
		if !v.applyRule(elementFieldName, element, param, "") {
			return false
		}
	}

	return true
}

// addError adds a validation error
func (v *Validator) addError(field, message, value string) {
	v.errors[field] = ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}

// getResult returns the validation result
func (v *Validator) getResult() *ValidationResult {
	return &ValidationResult{
		IsValid: len(v.errors) == 0,
		Errors:  v.errors,
	}
}

// ValidateJSON validates JSON request body against a struct
func ValidateJSON(c *fiber.Ctx, target interface{}) *ValidationResult {
	// Parse JSON body
	if err := c.BodyParser(target); err != nil {
		validator := NewValidator()
		validator.addError("_body", "Invalid JSON format", "")
		return validator.getResult()
	}

	// Validate struct
	validator := NewValidator()
	return validator.ValidateStruct(target)
}

// ValidateQuery validates query parameters
func ValidateQuery(c *fiber.Ctx, rules map[string]string) *ValidationResult {
	validator := NewValidator()

	for field, rule := range rules {
		value := c.Query(field)
		validator.validateField(field, value, rule)
	}

	return validator.getResult()
}

// ValidateParams validates URL parameters
func ValidateParams(c *fiber.Ctx, rules map[string]string) *ValidationResult {
	validator := NewValidator()

	for field, rule := range rules {
		value := c.Params(field)
		validator.validateField(field, value, rule)
	}

	return validator.getResult()
}

// HandleValidationErrors handles validation errors and returns appropriate response
func HandleValidationErrors(c *fiber.Ctx, result *ValidationResult) error {
	if result.IsValid {
		return nil
	}

	// Convert validation errors to response format
	errorDetails := make(map[string]string)
	for field, validationError := range result.Errors {
		errorDetails[field] = validationError.Message
	}

	return ValidationErrorResponse(c, errorDetails)
}

// IsValidJSON checks if a string is valid JSON
func IsValidJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// SanitizeString removes potentially harmful characters from string
func SanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	return input
}

// ValidateAndSanitizeInput validates and sanitizes input string
func ValidateAndSanitizeInput(input string, maxLength int) (string, error) {
	// Sanitize first
	sanitized := SanitizeString(input)

	// Check length
	if len(sanitized) > maxLength {
		return "", fmt.Errorf("input exceeds maximum length of %d characters", maxLength)
	}

	return sanitized, nil
}

// JSONMarshal is a custom JSON marshal function for Fiber
func JSONMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// JSONUnmarshal is a custom JSON unmarshal function for Fiber
func JSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ValidateStruct is a convenience function that validates a struct and returns an error
func ValidateStruct(s interface{}) error {
	validator := NewValidator()
	result := validator.ValidateStruct(s)

	if !result.IsValid {
		// Return the first error found
		for _, validationError := range result.Errors {
			return fmt.Errorf("validation failed for field '%s': %s", validationError.Field, validationError.Message)
		}
	}

	return nil
}
