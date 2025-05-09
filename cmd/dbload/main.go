// cmd/seed/main.go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

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
				// Try function call (must end with ())
				if strings.HasSuffix(valStr, ")") {
					result, isFunc, err := value.applyFunctionCall(valStr)
					if err != nil {
						return fmt.Errorf("function call error in %s: %w", k, err)
					}
					if isFunc {
						v = result
					}
				} else if strings.Contains(valStr, "|") {
					// Apply pipe functions if present
					result, err := value.applyPipes(valStr)
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
