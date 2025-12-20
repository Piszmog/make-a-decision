# Development Guide

This guide contains detailed information for developers who want to contribute to or modify the Make a Decision application.

## Technologies

This project uses the following technologies:

- [Go](https://golang.org/) - Backend language
- [templ](https://templ.guide/) - Type-safe HTML templating
- [HTMX](https://htmx.org/) - Dynamic HTML interactions
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework
- [sqlc](https://sqlc.dev/) - Type-safe SQL query generation
- [SQLite](https://www.sqlite.org/) - Database
- [golang migrate](https://github.com/golang-migrate/migrate) - Database migrations
- [air](https://github.com/air-verse/air) - Live reloading for development
- [playwright-go](https://github.com/playwright-community/playwright-go) - E2E testing

## Getting Started

### Prerequisites

- Go 1.21 or later
- [air](https://github.com/air-verse/air#installation) for live reloading

`templ`, `sqlc`, and `tailwindcss` (via [`go-tw`](https://github.com/Piszmog/go-tw)) are included as `go tool` directives. When running the application for the first time, it may take a little time as these tools are being downloaded and installed.

### Initial Setup

1. Clone the repository
2. Install dependencies:

```shell
go mod download
```

3. Generate code:

```shell
go tool sqlc generate
go tool templ generate -path ./internal/components
```

### Development Server

`air` provides live reloading during development. It watches for file changes and automatically rebuilds and restarts the application.

Install `air`:

```shell
go install github.com/air-verse/air@latest
```

Run the development server:

```shell
air
```

The application will start on http://localhost:8080/

The `.air.toml` configuration file defines the watch patterns and build commands.

## Build/Test Commands

- **Development**: `air` (live reload with templ/sqlc/tailwind generation)
- **Build**: `go build -o ./tmp/main ./cmd/server`
- **Lint**: `golangci-lint run` (Go linting with all enabled linters)
- **SQL Lint**: `go tool sqlc vet` (validates SQL queries)
- **Test all**: `go test -v ./...` (or `go test -race ./...` for race detection)
- **E2E tests**: `go test -v ./... -tags=e2e`
- **Single test**: `go test -v ./path/to/package -run TestName`
- **Generate**: `go tool templ generate -path ./internal/components && go tool sqlc generate`
- **CSS**: `go tool go-tw -i ./styles/input.css -o ./internal/dist/assets/css/output@dev.css`

## Project Structure

```text
.
├── cmd/
│   └── server/          # Application entrypoint
│       └── main.go
├── internal/            # Implementation code (not importable externally)
│   ├── components/      # templ HTML templates
│   │   ├── core/
│   │   └── home/
│   ├── db/              # Database layer
│   │   ├── migrations/  # SQL migration files
│   │   └── queries/     # SQL queries (sqlc generates Go code from these)
│   ├── dist/            # Embedded static assets
│   │   └── assets/
│   ├── log/             # Logging utilities
│   ├── server/          # HTTP server implementation
│   │   ├── handler/     # HTTP handlers
│   │   ├── middleware/  # HTTP middleware
│   │   └── router/      # Route definitions
│   └── version/         # Build version information
├── e2e/                 # End-to-end tests
├── styles/              # CSS source files
├── docs/                # Documentation assets
└── .air.toml            # Air configuration
```

### Why `internal/`?

All application code lives in `internal/` following Go's official server project layout:
- Prevents external packages from importing implementation details
- Signals this is a server application, not a reusable library
- Follows [go.dev/doc/modules/layout](https://go.dev/doc/modules/layout) "Server project" pattern
- `cmd/server/` contains the application entrypoint
- Only `e2e/` (tests) and `styles/` (build inputs) stay at root

## Code Style

- **Imports**: Standard library first, then third-party, then local packages
- **Naming**: Use Go conventions (PascalCase for exported, camelCase for unexported)
- **Error handling**: Always check errors, use `fmt.Errorf` with `%w` for wrapping
- **Logging**: Use structured logging with `slog.Logger`, include context in error messages
- **Interfaces**: Keep small and focused (e.g., `Database` interface in `internal/db/db.go`)
- **Comments**: Document exported functions/types, use `//` for single line comments

## Components (templ)

`templ` files live in `internal/components/`. These files define HTML templates with type safety.

### Templ Syntax

- **Components**: `templ ComponentName(params) { <html>content</html> }`
- **Expressions**: Use `{ variable }` for interpolation, `{ function() }` for function calls
- **Composition**: Call other components with `@ComponentName(args)`
- **All tags must be closed**: Use `<div></div>` or `<br/>` (self-closing)
- **Parameters**: Accept Go types as parameters: `templ Button(text string, disabled bool)`
- **File structure**: Package declaration, imports, then templ components
- **Generated files**: `*.go` files are auto-generated from `*.templ` files (ignored by git)

Example:

```templ
package components

templ HelloWorld(name string) {
  <div class="greeting">
    <h1>Hello, { name }!</h1>
  </div>
}
```

## Database (sqlc)

The database layer uses `sqlc` to generate type-safe Go code from SQL queries.

### SQLC Usage

- **Query annotations**: `-- name: FunctionName :one|:many|:exec` (required for all queries)
- **Return types**: `:one` (single row), `:many` (slice), `:exec` (error only), `:execresult` (sql.Result)
- **Parameters**: Use `?` for SQLite placeholders in queries
- **Generated code**: Run `go tool sqlc generate` to create Go functions from SQL
- **File structure**: Queries in `internal/db/queries/`, migrations in `internal/db/migrations/`
- **Usage pattern**: `queries := db.New(sqlDB); result, err := queries.FunctionName(ctx, params)`

### Migrations

This project uses [golang migrate](https://github.com/golang-migrate/migrate) for database migrations. Migrations are automatically run on application startup via `db.Migrate()` in `cmd/server/main.go`.

To create a new migration:

```shell
migrate create -ext sql -dir internal/db/migrations <name_of_migration>
```

This creates two files:
- `YYYYMMDDHHMMSS_name_of_migration.up.sql` - Applied when migrating up
- `YYYYMMDDHHMMSS_name_of_migration.down.sql` - Applied when rolling back

### Example: Remote Database Connection (Turso)

You can connect to a remote database like [Turso](https://turso.tech/) by creating a struct that implements the `Database` interface:

```go
package db

import (
	"database/sql"
	"log/slog"

	"github.com/Piszmog/make-a-decision/internal/db/queries"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type RemoteDB struct {
	logger  *slog.Logger
	db      *sql.DB
	queries *queries.Queries
}

var _ Database = (*RemoteDB)(nil)

func (d *RemoteDB) DB() *sql.DB {
	return d.db
}

func (d *RemoteDB) Queries() *queries.Queries {
	return d.queries
}

func (d *RemoteDB) Logger() *slog.Logger {
	return d.logger
}

func (d *RemoteDB) Close() error {
	return d.db.Close()
}

func NewRemoteDB(logger *slog.Logger, name string, token string) (*RemoteDB, error) {
	db, err := sql.Open("libsql", "libsql://"+name+".turso.io?authToken="+token)
	if err != nil {
		return nil, err
	}
	return &RemoteDB{logger: logger, db: db, queries: queries.New(db)}, nil
}
```

## HTMX Patterns

HTMX enables dynamic HTML interactions without writing JavaScript.

- **Basic requests**: `hx-get="/path"`, `hx-post="/path"`, `hx-put="/path"`, `hx-delete="/path"`
- **Triggers**: `hx-trigger="click"` (default), `hx-trigger="change"`, `hx-trigger="keyup delay:500ms"`
- **Targets**: `hx-target="#result"`, `hx-target="closest tr"`, `hx-target="next .error"`
- **Swapping**: `hx-swap="innerHTML"` (default), `hx-swap="outerHTML"`, `hx-swap="afterend"`
- **Indicators**: Add `class="htmx-indicator"` to show/hide loading states
- **Forms**: Include form values automatically, use `hx-include` for additional inputs
- **Boosting**: `hx-boost="true"` converts links/forms to AJAX requests

To upgrade HTMX, use the provided script:

```shell
./upgrade_htmx.sh
```

## Styling (Tailwind CSS)

This project uses Tailwind CSS for styling with utility-first classes.

### Common Patterns

- **Utility-first**: Use small, single-purpose classes like `text-center`, `bg-blue-500`, `p-4`
- **Responsive**: Prefix utilities with breakpoints: `sm:text-left`, `md:flex`, `lg:grid-cols-3`
- **States**: Use state prefixes: `hover:bg-blue-700`, `focus:ring-2`, `disabled:opacity-50`
- **Spacing**: Use consistent scale: `p-4` (padding), `m-2` (margin), `gap-6` (gap)
- **Colors**: Use semantic names: `bg-red-500`, `text-gray-700`, `border-blue-200`
- **Layout**: Common patterns: `flex items-center justify-between`, `grid grid-cols-2 gap-4`
- **Typography**: Size and weight: `text-xl font-bold`, `text-sm text-gray-600`

### Custom Styles

Custom CSS should be added to `styles/input.css`. The Tailwind CLI will process this file and generate the output CSS in `internal/dist/assets/css/` (auto-generated, ignored by git).

## Static Assets

Static assets (JavaScript, images, CSS) are stored in `internal/dist/assets/` and embedded into the application binary using Go's `embed` package.

- CSS is auto-generated by Tailwind (ignored by git)
- JavaScript files (like HTMX) are committed to the repository
- All assets are served from the `/assets/` route

## Server Architecture

The HTTP server uses Go's standard library with a middleware chain pattern.

### Components

- **Server**: `internal/server/server.go` - Main server with graceful shutdown (SIGINT handling)
- **Router**: `internal/server/router/router.go` - Route definitions using `http.ServeMux`
- **Middleware**: `internal/server/middleware/` - HTTP middleware (logging, caching, etc.)
- **Handlers**: `internal/server/handler/` - Request handlers with dependency injection

### Middleware Chain

Middleware is applied in order:
1. Logging - Logs all requests
2. Caching - Sets cache headers for static assets
3. Custom middleware - Application-specific middleware

## Testing

### Unit Tests

```shell
go test -v ./...
```

Run with race detection:

```shell
go test -race ./...
```

### E2E Tests

End-to-end tests use Playwright for browser automation.

```shell
go test -v ./... -tags=e2e
```

E2E tests automatically:
- Start the application on a random port
- Seed the database using `e2e/testdata/seed.sql`
- Run browser tests
- Clean up after completion

Supported browsers (set via `BROWSER` env var):
- `chromium` (default)
- `firefox`
- `webkit`

## Environment Variables

- **PORT**: Server port (default: 8080)
- **LOG_LEVEL**: debug, info, warn, error (default: info)
- **LOG_OUTPUT**: text, json (default: text)
- **DB_URL**: Database file path (default: ./db.sqlite3)

## Versioning

The `internal/version/` package allows setting a version at build time.

Default version is `dev`. To set a specific version:

```shell
go build -o ./app -ldflags="-X github.com/Piszmog/make-a-decision/internal/version.Value=1.0.0" ./cmd/server
```

## CI/CD

### GitHub Workflows

The repository includes two GitHub Actions workflows:

1. **ci.yml** - Continuous Integration
   - Runs on pull requests and pushes to main
   - Lints code with `golangci-lint`
   - Runs all tests including E2E
   - Validates SQL queries with `sqlc vet`

2. **release.yml** - Release Automation
   - Triggered by pushing a version tag (e.g., `v1.0.0`)
   - Creates a GitHub Release
   - Builds binaries for multiple platforms using [GoReleaser](https://goreleaser.com/)
   - Publishes Docker image to GitHub Container Registry

### Creating a Release

1. Tag the release:
```shell
git tag v1.0.0
git push origin v1.0.0
```

2. GitHub Actions will automatically:
   - Build binaries for Linux, macOS, Windows
   - Create a GitHub Release with binaries attached
   - Build and push Docker image

## AI Agent Support

The `AGENTS.md` file at the project root provides context for AI coding assistants. It includes:
- Common commands and workflows
- Code style guidelines
- Technology-specific patterns
- Project architecture overview

This helps AI agents better understand the project structure and development practices.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting
5. Commit your changes
6. Push to your fork
7. Open a Pull Request

Ensure your code:
- Passes all tests (`go test -v ./...`)
- Passes linting (`golangci-lint run`)
- Follows the code style guidelines
- Includes tests for new functionality
