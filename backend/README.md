# Aegis v14 - Backend (Go)

> Quant Trading System Backend API

**Language**: Go 1.21+
**Framework**: Gin
**Database**: PostgreSQL 15+
**Cache**: Redis 7.0+

---

## 📋 Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+
- Redis 7.0+
- golang-migrate (for migrations)
- air (for hot reload, optional)

```bash
# Install tools
brew install go postgresql redis golang-migrate

# Install air (optional, for hot reload)
go install github.com/cosmtrek/air@latest

# Install wire (for dependency injection)
go install github.com/google/wire/cmd/wire@latest

# Install golangci-lint (for linting)
brew install golangci-lint
```

---

## 🚀 Quick Start

### 1. Database Setup

```bash
# Initialize database
make db-init

# Check permissions
make db-check
```

### 2. Install Dependencies

```bash
make deps
```

### 3. Run Application

```bash
# Development mode (hot reload)
make dev

# Or standard run
make run
```

### 4. Run Tests

```bash
make test
```

---

## 📁 Project Structure

```
backend/
├── cmd/                       # Application entry points
│   ├── api/                   # BFF API server
│   └── runtime/               # Runtime engine (future)
│
├── internal/                  # Private application code
│   ├── api/                   # API layer (handlers, middleware, router)
│   ├── control/               # Control layer (risk, monitoring)
│   ├── strategy/              # Strategy layer (universe, signals, etc.)
│   ├── runtime/               # Runtime layer (pricesync, exit, etc.)
│   ├── infra/                 # Infrastructure layer (external, db, cache)
│   ├── domain/                # Domain models and events
│   └── pkg/                   # Internal shared libraries
│
├── pkg/                       # Public libraries
├── migrations/                # Database migrations
├── configs/                   # Configuration files
├── scripts/                   # Utility scripts
└── tests/                     # Integration and E2E tests
```

---

## 🛠️ Development

### Build

```bash
make build
```

### Run with Hot Reload

```bash
make dev
```

### Run Tests

```bash
# All tests
make test

# With coverage
make test-coverage
```

### Linting & Formatting

```bash
# Format code
make fmt

# Run linter
make lint
```

---

## 🗄️ Database

### Initialize Database

```bash
# Run init scripts
make db-init
```

### Migrations

```bash
# Create a new migration
make migrate-create NAME=create_users_table

# Run migrations
make migrate-up

# Rollback last migration
make migrate-down
```

### Check Permissions

```bash
# Check database permissions
make db-check

# Fix permissions if needed
make db-fix
```

---

## 🐳 Docker

### Start Services

```bash
# Start PostgreSQL + Redis
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

---

## 📝 Configuration

Configuration files are in `configs/`:

- `config.yaml` - Base configuration
- `config.dev.yaml` - Development overrides
- `config.prod.yaml` - Production overrides

Environment variables can override config values using `.env` file.

---

## 🧪 Testing

### Unit Tests

```bash
make test
```

### Integration Tests

```bash
go test -v ./tests/integration/...
```

### E2E Tests

```bash
go test -v ./tests/e2e/...
```

---

## 📚 Documentation

- [Architecture Design](../docs/architecture/)
- [Module Catalog](../docs/modules/module-catalog.md)
- [Database Schema](../docs/database/schema.md)
- [API Documentation](../docs/api/) (TBD)

---

## 🔍 Troubleshooting

### Database Connection Error

```bash
# Check database status
psql -U aegis_v14 -d aegis_v14 -c "SELECT 1"

# Fix permissions
make db-fix
```

### Port Already in Use

```bash
# Find process using port 8099
lsof -i :8099

# Kill process
kill -9 <PID>
```

---

## 📦 Dependencies

Major dependencies:

- **gin-gonic/gin** - Web framework
- **jackc/pgx** - PostgreSQL driver
- **redis/go-redis** - Redis client
- **shopspring/decimal** - Decimal math
- **rs/zerolog** - Structured logging
- **google/wire** - Dependency injection
- **stretchr/testify** - Testing toolkit

---

## 🤝 Contributing

1. Create a feature branch
2. Write tests
3. Ensure `make test` and `make lint` pass
4. Commit with clear message
5. Open a pull request

---

**Version**: v14.0.0
**Last Updated**: 2026-01-14
