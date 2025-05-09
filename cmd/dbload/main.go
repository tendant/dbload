// cmd/seed/main.go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/tendant/dbload/pkg/value"
	"gopkg.in/yaml.v3"
)

// registerCustomFunctions registers additional custom functions
func registerCustomFunctions() {
	// Register a custom function to generate a date in the future
	value.RegisterFunction("future", func(args []string) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("future function requires exactly one argument (days)")
		}

		// Parse the number of days
		var days int
		if _, err := fmt.Sscanf(args[0], "%d", &days); err != nil {
			return nil, fmt.Errorf("future function requires a number: %w", err)
		}

		// Calculate the future date
		futureDate := time.Now().UTC().AddDate(0, 0, days)
		return futureDate.Format(time.RFC3339), nil
	})

	// Register a custom function to convert text to uppercase
	value.RegisterFunction("upper", func(args []string) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("upper function requires exactly one argument")
		}
		return strings.ToUpper(args[0]), nil
	})
}

func loadYAML(path string) (map[string][]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string][]map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	// Note: YAML parsing strips quotes from values, so we need to be careful
	// when evaluating values that might contain pipes or function calls.
	// The Eval function will handle this by checking for specific function names
	// and pipe characters.

	return out, nil
}

func insertTable(db *sql.DB, table string, rows []map[string]interface{}, dryRun bool) error {
	for _, row := range rows {
		columns := []string{}
		placeholders := []string{}
		values := []interface{}{}
		idx := 1
		for k, v := range row {
			if valStr, ok := v.(string); ok {
				// Check if this is a function call or a pipe expression
				isFunctionCall := strings.HasPrefix(valStr, "hash ") ||
					valStr == "now" ||
					valStr == "uuid" ||
					strings.HasPrefix(valStr, "future ") ||
					strings.HasPrefix(valStr, "upper ")

				// For pipes, we need to be careful about the format
				hasPipe := strings.Contains(valStr, "|")

				if isFunctionCall || hasPipe {
					// For debugging
					if dryRun {
						fmt.Printf("Evaluating: %s\n", valStr)
					}

					// If this is a pipe expression, we need to handle the first part specially
					// because YAML parsing strips quotes from values
					if hasPipe && !isFunctionCall {
						parts := strings.SplitN(valStr, "|", 2)
						if len(parts) == 2 {
							// Wrap the first part in quotes to make it a literal value
							valStr = fmt.Sprintf("'%s'|%s", parts[0], parts[1])
							if dryRun {
								fmt.Printf("Modified to: %s\n", valStr)
							}
						}
					}

					result, err := value.Eval(valStr)
					if err != nil {
						return fmt.Errorf("value evaluation error in %s: %w", k, err)
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

		if dryRun {
			// In dry run mode, print the SQL statement and values
			fmt.Printf("SQL: %s\n", sqlStmt)
			fmt.Printf("Values: %v\n", values)
			fmt.Println("---")
		} else {
			// In normal mode, execute the SQL statement
			_, err := db.Exec(sqlStmt, values...)
			if err != nil {
				return fmt.Errorf("insert into %s failed: %w", table, err)
			}
		}
	}
	return nil
}

func main() {
	// Register custom functions
	registerCustomFunctions()

	// Parse command line flags
	path := flag.String("file", "seed.yaml", "Path to YAML seed file")
	dryRun := flag.Bool("dry-run", false, "Print SQL statements without executing them")
	flag.Parse()

	// Only require DATABASE_URL if not in dry run mode
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" && !*dryRun {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required (or use --dry-run)")
		os.Exit(1)
	}

	// Open database connection if not in dry run mode
	var db *sql.DB
	var err error
	if !*dryRun {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			panic(err)
		}
		defer db.Close()
	}

	seedData, err := loadYAML(*path)
	if err != nil {
		panic(err)
	}

	for table, rows := range seedData {
		fmt.Printf("Processing table: %s (%d rows)\n", table, len(rows))
		if err := insertTable(db, table, rows, *dryRun); err != nil {
			panic(err)
		}
	}

	if *dryRun {
		fmt.Println("✅ Dry run completed successfully.")
	} else {
		fmt.Println("✅ Seed data loaded successfully.")
	}
}
