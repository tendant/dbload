// cmd/seed/main.go
package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

var fnCallPattern = regexp.MustCompile(`^(\w+)\((.*?)\)$`)

func loadYAML(path string) (map[string][]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string][]map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

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

func insertTable(db *sql.DB, table string, rows []map[string]interface{}) error {
	for _, row := range rows {
		columns := []string{}
		placeholders := []string{}
		values := []interface{}{}
		idx := 1
		for k, v := range row {
			if valStr, ok := v.(string); ok {
				// Try function call (must end with ())
				if strings.HasSuffix(valStr, ")") {
					result, isFunc, err := applyFunctionCall(valStr)
					if err != nil {
						return fmt.Errorf("function call error in %s: %w", k, err)
					}
					if isFunc {
						v = result
					}
				} else if strings.Contains(valStr, "|") {
					// Apply pipe functions if present
					result, err := applyPipes(valStr)
					if err != nil {
						return fmt.Errorf("pipe error in %s: %w", k, err)
					}
					v = result
				}
			}

			columns = append(columns, k)
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
			values = append(values, v)
			idx++
		}

		sqlStmt := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
			table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)

		_, err := db.Exec(sqlStmt, values...)
		if err != nil {
			return fmt.Errorf("insert into %s failed: %w", table, err)
		}
	}
	return nil
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	path := flag.String("file", "seed.yaml", "Path to YAML seed file")
	flag.Parse()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	seedData, err := loadYAML(*path)
	if err != nil {
		panic(err)
	}

	for table, rows := range seedData {
		fmt.Printf("Seeding table: %s (%d rows)\n", table, len(rows))
		if err := insertTable(db, table, rows); err != nil {
			panic(err)
		}
	}

	fmt.Println("✅ Seed data loaded successfully.")
}
