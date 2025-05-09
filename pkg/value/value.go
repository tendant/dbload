package value

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	// Register the hash function
	RegisterFunction("hash", func(args []string) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("hash function requires exactly one argument, got %d", len(args))
		}
		h := sha256.Sum256([]byte(args[0]))
		return hex.EncodeToString(h[:]), nil
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

// Eval evaluates a string value according to the specified rules:
// 1. Literal values are quoted using single or double quotes
// 2. String can be separated as multiple parts using pipe '|'
// 3. Each part can be a literal value (if quoted) or a function call (if not quoted)
// 4. For function calls, the first token is the function name, and the rest are arguments
// 5. If there is a part before a function call, the previous part's value will be the last argument of the next function call
func Eval(value string) (interface{}, error) {
	parts := strings.Split(value, "|")
	var result interface{}

	for i, part := range parts {
		part = strings.TrimSpace(part)

		// Check if this is a quoted literal value
		matches := quotePattern.FindStringSubmatch(part)
		if matches != nil && len(matches) == 4 && matches[1] == matches[3] {
			// It's a quoted literal value
			result = matches[2]
			continue
		}

		// It's a function call
		tokens := strings.Fields(part)
		if len(tokens) == 0 {
			return nil, fmt.Errorf("empty function call")
		}

		fn := tokens[0]
		args := tokens[1:]

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
