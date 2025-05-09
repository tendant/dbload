# DBLoad

A utility for loading data into a database from YAML files with support for dynamic value generation.

## Features

- Load data from YAML files into database tables
- Support for dynamic value generation using functions
- Customizable function registry for extending functionality
- Support for function chaining using pipes

## Installation

```bash
go get github.com/tendant/dbload
```

## Usage

### Basic Usage

```bash
# Set your database connection string
export DATABASE_URL="postgres://user:password@localhost:5432/dbname"

# Run the tool with your YAML file
dbload -file seed.yaml

# Dry run mode (prints SQL without executing)
dbload -file seed.yaml -dry-run
```

### Command Line Options

- `-file`: Path to the YAML seed file (default: "seed.yaml")
- `-dry-run`: Print SQL statements without executing them (doesn't require DATABASE_URL)

### YAML File Format

```yaml
table_name:
  - column1: value1
    column2: value2
    column3: function_name(arg1, arg2)
    column4: 'literal value'
    column5: value|function_name()
```

## Value Functions

Values in the YAML file can use functions for dynamic value generation. There are two ways to use functions:

1. **Direct function calls**: `function_name(arg1, arg2, ...)`
2. **Piped functions**: `value|function_name()`

Functions are explicitly identified by parentheses, making it clear what is a function call and what is a literal value. Arguments are comma-separated within the parentheses.

### Built-in Functions

- `hash`: Generates a SHA-256 hash of the input
  - Example: `hash(password123)`
  
- `bcrypt`: Generates a secure bcrypt hash for password storage
  - Example: `bcrypt(password123)` or `bcrypt(password123, 12)` (with custom cost)
  - The second argument is optional and specifies the cost factor (default is 10)
  - Recommended for password storage as it's more secure than SHA-256
  
- `now`: Generates the current timestamp in RFC3339 format
  - Example: `now()`
  
- `uuid`: Generates a random UUID
  - Example: `uuid()`

### Custom Functions

The example includes two custom functions:

- `future`: Generates a date in the future by adding days to the current date
  - Example: `future(30)` (30 days from now)
  
- `upper`: Converts text to uppercase
  - Example: `upper(hello)` or `hello|upper()`

### Literal Values

Literal values can be quoted using single or double quotes:

- Example with single quotes: `'literal value'`
- Example with double quotes: `"literal value"`

## Extending with Custom Functions

You can register your own custom functions:

```go
import "github.com/tendant/dbload/pkg/value"

func init() {
    // Register a custom function
    value.RegisterFunction("myfunction", func(args []string) (interface{}, error) {
        // Validate arguments
        if len(args) != 1 {
            return nil, fmt.Errorf("myfunction requires exactly one argument")
        }
        
        // Process the argument
        result := processArg(args[0])
        
        // Return the result
        return result, nil
    })
}
```

## Example

See the `example.yaml` file for examples of using both built-in and custom functions.

## License

[License information]
