# Protoconf Development Guide

## Build & Test Commands
- Build all packages: `go build -v ./...`
- Run all tests: `go test -v ./...`
- Run single test: `go test -v github.com/protoconf/protoconf/path/to/package -run TestName`
- Run with coverage: `go test -race -coverprofile=coverage.txt -covermode=atomic ./...`
- Compile config: `protoconf compile "path/to/protoconf" config/path.pconf`
- Run agent: `protoconf agent path/to/protoconf/`

## Code Style Guidelines
- Follow standard Go conventions (gofmt)
- Use meaningful variable names (camelCase for variables, PascalCase for exported)
- Imports ordered: standard library, external packages, internal packages
- Properly handle errors (no ignored errors, provide context)
- Use context for cancellation and timeouts
- Prefer explicit type declarations over `:=` for clarity
- Comments should explain "why" not just "what"
- Add tests for new functionality
- Proto files should use snake_case for field names

## Project Structure
- `/cmd/` - Main applications
- `/agent/` - Protoconf agent code
- `/compiler/` - Starlark compilation logic
- `/server/` - gRPC server implementation
- Use `.pconf` for config files, `.pinc` for importable Starlark modules