# DBLoad

A utility for loading data into a database from YAML files with support for dynamic value generation.

## Features

- Load data from YAML files into database tables
- Support for dynamic value generation using functions
- Customizable function registry for extending functionality
- Support for function chaining using pipes

## Installation

You can install the tool using either `go get` or `go install`:

```bash
# Using go get (for adding to your project)
go get github.com/tendant/dbload

# Using go install (for installing the command-line tool)
go install github.com/tendant/dbload/cmd/dbload@latest
```

After installation with `go install`, the `dbload` command will be available in your PATH.

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
  
- `uuid`: Generates a UUID
  - Example: `uuid()` (random UUID) or `uuid(seed)` (deterministic UUID based on seed)
  - When a seed is provided, the same seed will always generate the same UUID
  - This is useful for referencing the same entity across different tables

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

## Referencing Data Between Tables

When loading data into multiple tables with relationships, you often need to reference data from one table in another. Here are some approaches to handle this:

### Using Fixed IDs

The simplest approach is to use fixed IDs in your YAML file, ensuring that the referenced IDs exist in the related tables:

```yaml
# Define users with known IDs
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"

# Reference user ID in orders
orders:
  - id: 101
    user_id: 1  # References user with ID 1
    product: "Laptop"
    quantity: 2
```

### Using UUID Function with Seeds

For more dynamic references, you can use the UUID function with a fixed seed to generate consistent IDs across tables:

```yaml
# First table with UUID-based ID
products:
  - id: "uuid(product-1)"  # This generates a consistent UUID based on "product-1"
    name: "Laptop"
    price: 999.99

# Reference the product in another table
order_items:
  - order_id: 1
    product_id: "uuid(product-1)"  # Same UUID as above
    quantity: 2
```

The UUID function with a seed will always generate the same UUID for the same seed value, making it perfect for maintaining referential integrity across tables without having to use sequential IDs.

### Proposed Enhancement: Reference Function

A more robust solution would be to add a `ref` function that can look up values from previously inserted rows:

```yaml
# First table
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"

# Reference data from the users table
orders:
  - id: 101
    user_id: "ref(users, 1, id)"  # References id column from users table where id=1
    user_email: "ref(users, 1, email)"  # References email column
```

This enhancement would require tracking inserted rows and their values during the insertion process.

## Testing with Sample Database

To test the tool with the provided example.yaml file, you can create the following sample database tables:

```sql
-- Create users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP,
    role VARCHAR(50),
    created_at TIMESTAMP,
    status VARCHAR(20)
);

-- Create products table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50),
    sku VARCHAR(100) UNIQUE,
    description TEXT,
    price DECIMAL(10, 2),
    created_at TIMESTAMP
);

-- Create inventory table (demonstrates relationships)
CREATE TABLE inventory (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id),
    product_sku VARCHAR(100) REFERENCES products(sku),
    warehouse VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL,
    last_updated TIMESTAMP
);
```

You can execute these SQL statements in your PostgreSQL database before running the tool. Then use the following commands to test:

```bash
# Set your database connection string
export DATABASE_URL="postgres://user:password@localhost:5432/yourdb"

# Run in dry-run mode first to see what would be inserted
dbload -file example.yaml -dry-run

# Then run for real to insert the data
dbload -file example.yaml
```

After running, you can verify the data was inserted correctly:

```sql
-- Check users table
SELECT * FROM users;

-- Check products table
SELECT * FROM products;

-- Check inventory table and its relationships
SELECT i.*, p.name AS product_name
FROM inventory i
LEFT JOIN products p ON i.product_id = p.id OR i.product_sku = p.sku;
```

## License

[License information]
