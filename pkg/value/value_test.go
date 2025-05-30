package value

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestCustomFunctions(t *testing.T) {
	// Register a custom function for testing
	RegisterFunction("double", func(args []string) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("double function requires exactly one argument")
		}
		return args[0] + args[0], nil
	})
	// Make sure to clean up after the test
	defer UnregisterFunction("double")

	// Test the custom function
	result, err := Eval("double(test)")
	if err != nil {
		t.Errorf("Eval() error = %v", err)
		return
	}
	expected := "testtest"
	if result != expected {
		t.Errorf("Eval() = %v, want %v", result, expected)
	}

	// Test the custom function in a pipe
	result, err = Eval(`value|double()`)
	if err != nil {
		t.Errorf("Eval() error = %v", err)
		return
	}
	expected = "valuevalue"
	if result != expected {
		t.Errorf("Eval() = %v, want %v", result, expected)
	}

	// Test unregistering a function
	UnregisterFunction("double")
	_, err = Eval("double(test)")
	if err == nil {
		t.Errorf("Expected error after unregistering function, got nil")
	}
}

func TestEval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(t *testing.T, result interface{})
	}{
		// Basic literal values
		{
			name:    "Simple literal",
			input:   "test value",
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				if result != "test value" {
					t.Errorf("Expected 'test value', got %v", result)
				}
			},
		},
		{
			name:    "Quoted literal",
			input:   `"test value"`,
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				if result != `"test value"` {
					t.Errorf("Expected '\"test value\"', got %v", result)
				}
			},
		},

		// Single function calls
		{
			name:    "Hash function with argument",
			input:   "hash(test)",
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				expected := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" // SHA-256 hash of "test"
				if result != expected {
					t.Errorf("Expected hash '%s', got '%v'", expected, result)
				}
			},
		},
		{
			name:    "Hash function with quoted argument",
			input:   `hash("test")`,
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				expected := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" // SHA-256 hash of "test"
				if result != expected {
					t.Errorf("Expected hash '%s', got '%v'", expected, result)
				}
			},
		},
		{
			name:    "Bcrypt function with default cost",
			input:   "bcrypt(password123)",
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				str, ok := result.(string)
				if !ok {
					t.Errorf("Expected string result, got %T", result)
					return
				}

				// Verify it's a valid bcrypt hash
				if len(str) < 60 || !strings.HasPrefix(str, "$2a$") {
					t.Errorf("Result is not a valid bcrypt hash: %s", str)
					return
				}

				// Verify the hash works for the original password
				err := bcrypt.CompareHashAndPassword([]byte(str), []byte("password123"))
				if err != nil {
					t.Errorf("Bcrypt hash verification failed: %v", err)
				}
			},
		},
		{
			name:    "Bcrypt function with custom cost",
			input:   "bcrypt(password123, 12)",
			wantErr: false,
			validate: func(t *testing.T, result interface{}) {
				str, ok := result.(string)
				if !ok {
					t.Errorf("Expected string result, got %T", result)
					return
				}

				// Verify it's a valid bcrypt hash with cost 12
				if len(str) < 60 || !strings.HasPrefix(str, "$2a$12$") {
					t.Errorf("Result is not a valid bcrypt hash with cost 12: %s", str)
					return
				}

				// Verify the hash works for the original password
				err := bcrypt.CompareHashAndPassword([]byte(str), []byte("password123"))
				if err != nil {
					t.Errorf("Bcrypt hash verification failed: %v", err)
				}
			},
		},
		{
			name:    "UUID function without arguments",
			input:   "uuid()",
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
			name:    "UUID function with seed",
			input:   "uuid(test-seed)",
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

				// Run the function again with the same seed to verify consistency
				result2, err := Eval("uuid(test-seed)")
				if err != nil {
					t.Errorf("Second Eval() error = %v", err)
					return
				}

				// Verify that the UUIDs are the same
				if result != result2 {
					t.Errorf("UUID with same seed produced different results: %v vs %v", result, result2)
				}
			},
		},
		{
			name:    "Now function without arguments",
			input:   "now()",
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
			input:   `value|hash()`,
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
			input:   `hash(test)|hash()`,
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
			name:    "Empty pipe",
			input:   "|",
			wantErr: false, // This is now just a literal "|"
			validate: func(t *testing.T, result interface{}) {
				if result != "" {
					t.Errorf("Expected empty string, got %v", result)
				}
			},
		},
		{
			name:    "Unsupported function",
			input:   "invalid(test)",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Hash with no arguments",
			input:   "hash()",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Hash with too many arguments",
			input:   "hash(arg1, arg2)",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Bcrypt with too many arguments",
			input:   "bcrypt(arg1, 10, extra)",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Bcrypt with invalid cost",
			input:   "bcrypt(password, invalid)",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "UUID with too many arguments",
			input:   "uuid(arg1, arg2)",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Now with arguments",
			input:   "now(arg)",
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Hash to UUID pipe (UUID uses hash result as seed)",
			input:   "hash(test)|uuid()",
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

				// Verify that the UUID is deterministic based on the hash
				hashResult := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" // SHA-256 hash of "test"
				expectedUUID, err := Eval("uuid(" + hashResult + ")")
				if err != nil {
					t.Errorf("Failed to generate expected UUID: %v", err)
					return
				}

				if str != expectedUUID {
					t.Errorf("UUID from pipe doesn't match expected UUID: %v vs %v", str, expectedUUID)
				}
			},
		},
		{
			name:    "Literal to Now pipe (Now gets argument)",
			input:   `value|now()`,
			wantErr: true,
			validate: func(t *testing.T, result interface{}) {
				// No validation needed for error case
			},
		},
		{
			name:    "Malformed function call",
			input:   "hash(test",
			wantErr: false, // This is now just a literal "hash(test"
			validate: func(t *testing.T, result interface{}) {
				if result != "hash(test" {
					t.Errorf("Expected 'hash(test', got %v", result)
				}
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
