package value

import (
	"regexp"
	"testing"
	"time"
)

func TestEval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(t *testing.T, result interface{})
	}{
		// Basic literal values
		{
			name:    "Double quoted literal",
			input:   `"test value"`,
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				if result != "test value" {
					t.Errorf("Expected 'test value', got %v", result)
				}
			},
		},
		{
			name:    "Single quoted literal",
			input:   `'test value'`,
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				if result != "test value" {
					t.Errorf("Expected 'test value', got %v", result)
				}
			},
		},

		// Single function calls
		{
			name:    "Hash function with argument",
			input:   "hash test",
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				expected := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" // SHA-256 hash of "test"
				if result != expected {
					t.Errorf("Expected hash '%s', got '%v'", expected, result)
				}
			},
		},
		{
			name:    "UUID function without arguments",
			input:   "uuid",
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				str, ok := result.(string)
				if !ok {
					t.Errorf("Expected string result, got %T", result)
					return
				}

				// UUID format validation
				uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
				if !uuidPattern.MatchString(str) {
					t.Errorf("Result is not a valid UUID: %s", str)
				}
			},
		},
		{
			name:    "Now function without arguments",
			input:   "now",
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				str, ok := result.(string)
				if !ok {
					t.Errorf("Expected string result, got %T", result)
					return
				}

				// Parse the time to validate format
				_, err := time.Parse(time.RFC3339, str)
				if err != nil {
					t.Errorf("Result is not a valid RFC3339 time: %s, error: %v", str, err)
				}
			},
		},

		// Piped operations
		{
			name:    "Literal to hash",
			input:   `"value"|hash`,
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				expected := "cd42404d52ad55ccfa9aca4adc828aa5800ad9d385a0671fbcbf724118320619" // SHA-256 hash of "value"
				if result != expected {
					t.Errorf("Expected hash '%s', got '%v'", expected, result)
				}
			},
		},
		{
			name:    "Double hash",
			input:   `hash test|hash`,
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				// First hash: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
				expected := "7b3d979ca8330a94fa7e9e1b466d8b99e0bcdea1ec90596c0dcc8d7ef6b4300c" // SHA-256 hash of the first hash
				if result != expected {
					t.Errorf("Expected double hash '%s', got '%v'", expected, result)
				}
			},
		},

		// Error cases
		{
			name:    "Empty function call",
			input:   "|",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Unsupported function",
			input:   "invalid test",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Hash with no arguments",
			input:   "hash",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Hash with too many arguments",
			input:   "hash arg1 arg2",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "UUID with arguments",
			input:   "uuid arg",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Now with arguments",
			input:   "now arg",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Hash to UUID pipe (UUID gets argument)",
			input:   "hash test|uuid",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Literal to Now pipe (Now gets argument)",
			input:   `"value"|now`,
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Unclosed quote",
			input:   `"unclosed`,
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Mismatched quotes",
			input:   `"mismatched'`,
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Skip validation if we expected an error
			if tt.wantErr {
				return
			}

			// Validate the result
			tt.validate(t, result)
		})
	}
}
