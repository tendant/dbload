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

// loadYAML loads data from a YAML file and returns both the data and the order of tables
func loadYAML(path string) (map[string][]map[string]interface{}, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	// First, unmarshal into a yaml.Node to preserve order
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, nil, err
	}

	// Then unmarshal into our map for easier access
	var out map[string][]map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, nil, err
	}

	// Extract the order of tables from the yaml.Node
	var tableOrder []string
	if len(root.Content) > 0 && root.Content[0].Kind == yaml.MappingNode {
		mapping := root.Content[0]
		// In a mapping node, keys are at even indices (0, 2, 4, ...)
		for i := 0; i < len(mapping.Content); i += 2 {
			if mapping.Content[i].Kind == yaml.ScalarNode {
				tableName := mapping.Content[i].Value
				tableOrder = append(tableOrder, tableName)
			}
		}
	}

	// Note: YAML parsing strips quotes from values, so we need to be careful
	// when evaluating values that might contain pipes or function calls.
	// The Eval function will handle this by checking for specific function names
	// and pipe characters.

	return out, tableOrder, nil
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
				isFunctionCall := strings.Contains(valStr, "(") && strings.Contains(valStr, ")")
				hasPipe := strings.Contains(valStr, "|")

				if isFunctionCall || hasPipe {
					// For debugging
					if dryRun {
						fmt.Printf("Evaluating: %s\n", valStr)
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
	orderStr := flag.String("order", "", "Comma-separated list of table names to specify insertion order")
	respectYamlOrder := flag.Bool("respect-yaml-order", true, "Process tables in the order they appear in the YAML file")
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

	seedData, tableOrder, err := loadYAML(*path)
	if err != nil {
		panic(err)
	}

	// Process tables in specified order if provided via command line
	if *orderStr != "" {
		// Command line order takes precedence
		tableOrder = strings.Split(*orderStr, ",")
		for i, table := range tableOrder {
			tableOrder[i] = strings.TrimSpace(table)
		}
		*respectYamlOrder = false // Override YAML order when explicit order is provided
	}

	// Process tables in the specified order
	if len(tableOrder) > 0 && (*respectYamlOrder || *orderStr != "") {
		for _, table := range tableOrder {
			if rows, ok := seedData[table]; ok {
				fmt.Printf("Processing table: %s (%d rows)\n", table, len(rows))
				if err := insertTable(db, table, rows, *dryRun); err != nil {
					panic(err)
				}
				// Remove the table from the map to avoid processing it again
				delete(seedData, table)
			} else {
				fmt.Printf("Warning: Table '%s' in order but not found in YAML data\n", table)
			}
		}
	}

	// Process any remaining tables not specified in the order
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
