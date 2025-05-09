package value

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var fnCallPattern = regexp.MustCompile(`^(\w+)\((.*?)\)$`)

func applyPipes(value string) (interface{}, error) {
	parts := strings.Split(value, "|")
	if len(parts) == 1 {
		return value, nil
	}

	input := strings.TrimSpace(parts[0])
	for _, fn := range parts[1:] {
		fn = strings.TrimSpace(fn)
		switch fn {
		case "hash":
			h := sha256.Sum256([]byte(input))
			input = hex.EncodeToString(h[:])
		case "now":
			input = time.Now().UTC().Format(time.RFC3339)
		case "uuid":
			input = uuid.New().String()
		default:
			return nil, fmt.Errorf("unsupported function: %s", fn)
		}
	}

	return input, nil
}

func applyFunctionCall(value string) (interface{}, bool, error) {
	matches := fnCallPattern.FindStringSubmatch(value)
	if len(matches) != 3 {
		return value, false, nil
	}
	fn := matches[1]
	arg := strings.Trim(matches[2], `"'`)

	switch fn {
	case "hash":
		h := sha256.Sum256([]byte(arg))
		return hex.EncodeToString(h[:]), true, nil
	case "now":
		return time.Now().UTC().Format(time.RFC3339), true, nil
	case "uuid":
		return uuid.New().String(), true, nil
	default:
		return nil, true, fmt.Errorf("unsupported function: %s", fn)
	}
}
