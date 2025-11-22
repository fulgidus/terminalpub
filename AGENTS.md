# Agent Guidelines - terminalpub

## Build & Test Commands
- `make build` - Build the binary
- `make test` - Run all tests
- `go test ./internal/services/... -run TestPostService` - Run a single test
- `make lint` - Run golangci-lint
- `make format` - Format code with gofmt and goimports
- `make migrate-up` - Run database migrations
- `make docker-up` - Start PostgreSQL & Redis via Docker

## Code Style
- **Imports**: Standard library first, then external, then internal (use `goimports`)
- **Formatting**: Run `gofmt` before committing (enforced by CI)
- **Types**: Prefer explicit types; use structs for complex data; leverage Go generics where appropriate
- **Naming**: Use camelCase for unexported, PascalCase for exported; descriptive variable names; no single-letter vars except loop indices
- **Error Handling**: Always check errors; wrap with context using `fmt.Errorf("context: %w", err)`; return errors up the stack
- **Comments**: All exported functions/types must have godoc comments starting with the name
- **Testing**: Table-driven tests preferred; use `_test.go` suffix; mock external dependencies
- **Commits**: Conventional commits format - `feat:`, `fix:`, `docs:`, `refactor:`, `test:`
- **Structure**: Follow `/internal` for private packages, `/cmd` for binaries, `/pkg` for public libraries

## Key Conventions
- Database models in `internal/models/`; use GORM or pgx for queries
- ActivityPub logic in `internal/activitypub/`; validate all incoming JSON
- TUI components in `internal/ui/`; follow Bubbletea patterns (Model/Update/View)
