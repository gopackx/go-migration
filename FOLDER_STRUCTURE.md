# Go Migrator - Folder Structure

## Complete Project Structure

```
go-migrator/
│
├── cmd/
│   └── migrator/                      # CLI Entry Point
│       ├── main.go                    # Main application
│       └── version.go                 # Version info
│
├── pkg/                               # Public API (dapat diimport package lain)
│   │
│   ├── migrator/                      # Core Migration Engine
│   │   ├── migrator.go               # Main migrator implementation
│   │   ├── interfaces.go             # Public interfaces
│   │   ├── migration.go              # Migration struct & base
│   │   ├── registry.go               # Migration registry
│   │   ├── runner.go                 # Migration execution engine
│   │   ├── tracker.go                # Database migration tracking
│   │   ├── batch.go                  # Batch management
│   │   ├── hooks.go                  # Before/After hooks
│   │   └── errors.go                 # Custom errors
│   │
│   ├── schema/                        # Schema Builder (Laravel-like)
│   │   ├── schema.go                 # Schema interface implementation
│   │   ├── blueprint.go              # Table blueprint
│   │   ├── column.go                 # Column definitions
│   │   ├── index.go                  # Index builder
│   │   ├── foreign_key.go            # Foreign key builder
│   │   ├── grammar.go                # Base SQL grammar
│   │   │
│   │   └── grammars/                 # Database-specific SQL generators
│   │       ├── postgres.go           # PostgreSQL grammar
│   │       ├── mysql.go              # MySQL grammar
│   │       ├── sqlite.go             # SQLite grammar
│   │       └── sqlserver.go          # SQL Server grammar (optional)
│   │
│   ├── seeder/                        # Seeding System
│   │   ├── seeder.go                 # Seeder interface
│   │   ├── runner.go                 # Seeder execution engine
│   │   ├── registry.go               # Seeder registry
│   │   ├── interfaces.go             # Public interfaces
│   │   │
│   │   └── factory/                  # Factory Pattern
│   │       ├── factory.go            # Factory implementation
│   │       ├── builder.go            # Fluent builder
│   │       ├── faker.go              # Faker interface
│   │       ├── faker_impl.go         # Faker implementation
│   │       └── state.go              # Factory states
│   │
│   ├── database/                      # Database Connection Management
│   │   ├── connection.go             # Connection interface
│   │   ├── manager.go                # Multi-connection manager
│   │   ├── pool.go                   # Connection pooling
│   │   ├── transaction.go            # Transaction wrapper
│   │   │
│   │   └── drivers/                  # Database Drivers
│   │       ├── driver.go             # Driver interface
│   │       ├── postgres.go           # PostgreSQL driver
│   │       ├── mysql.go              # MySQL driver
│   │       └── sqlite.go             # SQLite driver
│   │
│   └── cli/                           # CLI Commands Package
│       ├── app.go                    # CLI application setup
│       ├── context.go                # CLI context
│       │
│       └── commands/                 # CLI Commands
│           ├── migrate.go            # migrate
│           ├── migrate_status.go     # migrate:status
│           ├── migrate_rollback.go   # migrate:rollback
│           ├── migrate_reset.go      # migrate:reset
│           ├── migrate_refresh.go    # migrate:refresh
│           ├── migrate_fresh.go      # migrate:fresh
│           ├── migrate_install.go    # migrate:install (create table)
│           ├── make_migration.go     # make:migration
│           ├── seed.go               # db:seed
│           ├── make_seeder.go        # make:seeder
│           └── root.go               # Root command
│
├── internal/                          # Private Implementation (tidak bisa diimport)
│   │
│   ├── config/                       # Configuration Management
│   │   ├── config.go                 # Config struct & loader
│   │   ├── parser.go                 # YAML/JSON parser
│   │   ├── validator.go              # Config validation
│   │   └── defaults.go               # Default values
│   │
│   ├── logger/                       # Logging System
│   │   ├── logger.go                 # Logger interface
│   │   ├── console.go                # Console logger
│   │   ├── file.go                   # File logger
│   │   └── formatter.go              # Log formatters
│   │
│   ├── scanner/                      # File Scanner
│   │   ├── migration_scanner.go      # Scan migration files
│   │   ├── seeder_scanner.go         # Scan seeder files
│   │   └── loader.go                 # Dynamic loading
│   │
│   ├── generator/                    # Code Generator
│   │   ├── migration.go              # Migration generator
│   │   ├── seeder.go                 # Seeder generator
│   │   ├── template.go               # Template engine
│   │   └── formatter.go              # Code formatter
│   │
│   └── utils/                        # Internal Utilities
│       ├── string.go                 # String helpers
│       ├── file.go                   # File helpers
│       ├── time.go                   # Time helpers
│       ├── reflect.go                # Reflection helpers
│       └── sql.go                    # SQL helpers
│
├── examples/                          # Usage Examples
│   │
│   ├── basic/                        # Basic usage example
│   │   ├── main.go
│   │   ├── migrations/
│   │   │   ├── 20240215000001_create_users_table.go
│   │   │   └── 20240215000002_create_posts_table.go
│   │   └── seeders/
│   │       ├── user_seeder.go
│   │       └── post_seeder.go
│   │
│   ├── advanced/                     # Advanced features
│   │   ├── main.go
│   │   ├── migrations/
│   │   │   ├── 20240215000001_create_multi_tenant.go
│   │   │   └── 20240215000002_add_indexes.go
│   │   ├── seeders/
│   │   │   └── tenant_seeder.go
│   │   └── factories/
│   │       └── factories.go
│   │
│   └── multi-database/               # Multiple database example
│       ├── main.go
│       ├── config.yaml
│       └── migrations/
│           ├── postgres/
│           └── mysql/
│
├── tests/                             # Test Suite
│   │
│   ├── unit/                         # Unit Tests
│   │   ├── migrator/
│   │   │   ├── migrator_test.go
│   │   │   ├── runner_test.go
│   │   │   └── tracker_test.go
│   │   ├── schema/
│   │   │   ├── blueprint_test.go
│   │   │   └── grammar_test.go
│   │   └── seeder/
│   │       ├── factory_test.go
│   │       └── faker_test.go
│   │
│   ├── integration/                  # Integration Tests
│   │   ├── postgres_test.go
│   │   ├── mysql_test.go
│   │   ├── sqlite_test.go
│   │   └── migration_flow_test.go
│   │
│   ├── e2e/                          # End-to-End Tests
│   │   ├── cli_test.go
│   │   └── workflow_test.go
│   │
│   ├── fixtures/                     # Test Fixtures
│   │   ├── migrations/
│   │   ├── seeders/
│   │   └── databases/
│   │
│   ├── mocks/                        # Mock Objects
│   │   ├── mock_db.go
│   │   ├── mock_migrator.go
│   │   └── mock_seeder.go
│   │
│   └── testutil/                     # Test Utilities
│       ├── database.go               # Test database setup
│       ├── assertions.go             # Custom assertions
│       └── helpers.go                # Test helpers
│
├── templates/                         # Code Generation Templates
│   ├── migration.tmpl                # Migration template
│   ├── migration_create.tmpl         # Create table migration
│   ├── migration_alter.tmpl          # Alter table migration
│   ├── seeder.tmpl                   # Seeder template
│   └── factory.tmpl                  # Factory template
│
├── docs/                              # Documentation
│   ├── getting-started.md
│   ├── migrations.md
│   ├── seeders.md
│   ├── schema-builder.md
│   ├── factories.md
│   ├── cli-commands.md
│   ├── configuration.md
│   ├── multi-database.md
│   ├── hooks.md
│   └── api-reference.md
│
├── scripts/                           # Build & Development Scripts
│   ├── build.sh                      # Build script
│   ├── test.sh                       # Test script
│   ├── release.sh                    # Release script
│   ├── install.sh                    # Installation script
│   └── dev-setup.sh                  # Dev environment setup
│
├── .github/                           # GitHub Configuration
│   ├── workflows/
│   │   ├── test.yml                  # CI testing
│   │   ├── lint.yml                  # Linting
│   │   ├── release.yml               # Release automation
│   │   └── codeql.yml                # Security scanning
│   │
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   └── feature_request.md
│   │
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── FUNDING.yml
│
├── deployments/                       # Deployment Configurations
│   ├── docker/
│   │   ├── Dockerfile
│   │   ├── Dockerfile.dev
│   │   └── docker-compose.yml
│   │
│   └── kubernetes/
│       ├── deployment.yaml
│       └── service.yaml
│
├── .gitignore                        # Git ignore file
├── .golangci.yml                     # Linter configuration
├── .editorconfig                     # Editor configuration
├── go.mod                            # Go modules
├── go.sum                            # Go dependencies checksum
├── Makefile                          # Build automation
├── LICENSE                           # License file
├── README.md                         # Main documentation
├── CHANGELOG.md                      # Version changelog
├── CONTRIBUTING.md                   # Contribution guidelines
├── CODE_OF_CONDUCT.md               # Code of conduct
├── config.example.yaml              # Example configuration
└── .goreleaser.yml                  # GoReleaser config
```

## Penjelasan Struktur

### 1. `/cmd` - Application Entry Points
Berisi executable programs. Setiap subdirectory adalah satu binary.

**Best Practice:**
- Keep it thin - hanya setup dan call ke pkg
- No business logic di sini
- Import dari `/pkg` dan `/internal`

### 2. `/pkg` - Public Library Code
Code yang boleh diimport oleh aplikasi lain.

**Best Practice:**
- Well-documented public APIs
- Stable interfaces
- Backward compatibility
- Minimal external dependencies

### 3. `/internal` - Private Application Code
Code yang tidak boleh diimport oleh aplikasi lain (Go enforced).

**Best Practice:**
- Implementation details
- Utilities khusus aplikasi ini
- Tidak perlu maintain backward compatibility

### 4. `/examples` - Contoh Penggunaan
Demonstrasi cara menggunakan package.

**Best Practice:**
- Self-contained examples
- Cover common use cases
- Include README di setiap example

### 5. `/tests` - Test Suite
Organized test files terpisah dari source code.

**Best Practice:**
- Unit tests: Fast, isolated
- Integration tests: Real database
- E2E tests: Full workflow

### 6. `/templates` - Code Generation
Template untuk generate code.

**Best Practice:**
- Use Go templates
- Parameterized
- Easy to customize

### 7. `/docs` - Documentation
Comprehensive documentation.

**Best Practice:**
- Markdown format
- Include examples
- Keep updated

## File Naming Conventions

```
# Migration files
YYYYMMDDHHMMSS_description.go
20240215120000_create_users_table.go

# Seeder files
name_seeder.go
user_seeder.go
post_seeder.go

# Test files
*_test.go
migrator_test.go

# Interface files
interfaces.go

# Implementation files
Descriptive names
postgres.go (not pg.go)
```

## Import Path Examples

```go
// Public API
import "github.com/yourusername/go-migrator/pkg/migrator"
import "github.com/yourusername/go-migrator/pkg/schema"
import "github.com/yourusername/go-migrator/pkg/seeder"

// Internal (only within go-migrator)
import "github.com/yourusername/go-migrator/internal/config"
import "github.com/yourusername/go-migrator/internal/logger"
```

## Keuntungan Struktur Ini

1. **Clear Separation**: Public vs Private code jelas terpisah
2. **Scalable**: Mudah tambah fitur baru
3. **Testable**: Test organization yang baik
4. **Go Standard**: Mengikuti Go project layout standard
5. **Enterprise Ready**: Suitable untuk production use
6. **Developer Friendly**: Easy to navigate dan understand
