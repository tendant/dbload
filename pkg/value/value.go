package value

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// QuotePattern matches single or double quoted strings
var quotePattern = regexp.MustCompile(`^(['"])(.*)(['"])$`)

// FunctionHandler defines the signature for custom functions
type FunctionHandler func(args []string) (interface{}, error)

// functionRegistry stores registered functions
var functionRegistry = map[string]FunctionHandler{}
var registryMutex sync.RWMutex

// RegisterFunction registers a custom function with the given name
func RegisterFunction(name string, handler FunctionHandler) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	functionRegistry[name] = handler
}

// UnregisterFunction removes a function from the registry
func UnregisterFunction(name string) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	delete(functionRegistry, name)
}

// GetFunction retrieves a function from the registry
func GetFunction(name string) (FunctionHandler, bool) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	handler, exists := functionRegistry[name]
	return handler, exists
}

// init registers the default functions
func init() {
	// Register the hash function (SHA-256)
	RegisterFunction("hash", func(args []string) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("hash function requires exactly one argument, got %d", len(args))
		}
		h := sha256.Sum256([]byte(args[0]))
		return hex.EncodeToString(h[:]), nil
	})

	// Register the bcrypt function for password hashing
	RegisterFunction("bcrypt", func(args []string) (interface{}, error) {
		// Check if we have 1 or 2 arguments (password, [cost])
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("bcrypt function requires 1 or 2 arguments (password, [cost]), got %d", len(args))
		}

		// Default cost is 10
		cost := bcrypt.DefaultCost

		// If cost is provided, parse it
		if len(args) == 2 {
			var err error
			cost, err = strconv.Atoi(args[1])
			if err != nil {
				return nil, fmt.Errorf("bcrypt cost must be a number: %w", err)
			}

			// Validate cost range
			if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
				return nil, fmt.Errorf("bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
			}
		}

		// Generate the hash
		hash, err := bcrypt.GenerateFromPassword([]byte(args[0]), cost)
		if err != nil {
			return nil, fmt.Errorf("bcrypt error: %w", err)
		}

		return string(hash), nil
	})

	// Register the now function
	RegisterFunction("now", func(args []string) (interface{}, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("now function requires no arguments, got %d", len(args))
		}
		return time.Now().UTC().Format(time.RFC3339), nil
	})

	// Register the uuid function
	RegisterFunction("uuid", func(args []string) (interface{}, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("uuid function requires no arguments, got %d", len(args))
		}
		return uuid.New().String(), nil
	})
}

// FunctionCallPattern matches function calls with parentheses: function(arg1, arg2, ...)
var functionCallPattern = regexp.MustCompile(`^(\w+)\((.*)\)$`)

// Eval evaluates a string value according to the specified rules:
// 1. String can be separated as multiple parts using pipe '|'
// 2. Each part can be a literal value or a function call
// 3. Function calls must use the syntax: function(arg1, arg2, ...)
// 4. If there is a part before a function call, the previous part's value will be the last argument of the next function call
func Eval(value string) (interface{}, error) {
	parts := strings.Split(value, "|")
	var result interface{}

	for i, part := range parts {
		part = strings.TrimSpace(part)

		// Check if this is a function call
		matches := functionCallPattern.FindStringSubmatch(part)
		if matches == nil || len(matches) != 3 {
			// It's a literal value
			result = part
			continue
		}

		// It's a function call
		fn := matches[1]
		argsStr := matches[2]

		// Parse arguments - split by comma, but respect quotes
		var args []string
		if argsStr != "" {
			// Simple argument parsing - split by comma and trim spaces
			args = strings.Split(argsStr, ",")
			for i, arg := range args {
				args[i] = strings.TrimSpace(arg)

				// Remove quotes if present
				if len(args[i]) > 1 {
					if (args[i][0] == '"' && args[i][len(args[i])-1] == '"') ||
						(args[i][0] == '\'' && args[i][len(args[i])-1] == '\'') {
						args[i] = args[i][1 : len(args[i])-1]
					}
				}
			}
		}

		// If there was a previous result and this isn't the first part,
		// add it as an argument
		if i > 0 && result != nil {
			resultStr, ok := result.(string)
			if !ok {
				resultStr = fmt.Sprintf("%v", result)
			}
			args = append(args, resultStr)
		}

		// Look up the function in the registry
		handler, exists := GetFunction(fn)
		if !exists {
			return nil, fmt.Errorf("unsupported function: %s", fn)
		}

		// Call the function handler
		var err error
		result, err = handler(args)
		if err != nil {
			return nil, fmt.Errorf("function %s error: %w", fn, err)
		}
	}

	return result, nil
}
