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
	return out, nil
}

func insertTable(db *sql.DB, table string, rows []map[string]interface{}) error {
	for _, row := range rows {
		columns := []string{}
		placeholders := []string{}
		values := []interface{}{}
		idx := 1
		for k, v := range row {
			if valStr, ok := v.(string); ok {
				// Use the new Eval function to handle both function calls and pipes
				result, err := value.Eval(valStr)
				if err != nil {
					return fmt.Errorf("value evaluation error in %s: %w", k, err)
				}
				v = result
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
	// Register custom functions
	registerCustomFunctions()

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
