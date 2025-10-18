# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**New API** is a next-generation AI gateway and model management system. It's a hybrid architecture project combining:
- **Backend**: Go 1.25.1 with Gin framework (github.com/QuantumNous/new-api)
- **Frontend**: React 18.2.0 with Vite and Semi Design UI
- **Features**: Multi-channel API relay, model management, billing system, user authentication, WebSocket support

The project aggregates multiple AI providers (OpenAI, Claude, Gemini, etc.) and provides unified API compatibility, channel management with weighted random selection, and per-model billing.

## Quick Start Commands

### Backend (Go)
```bash
# Download dependencies
go mod download

# Run backend development server
go run main.go

# Build binary
go build -o new-api

# Using Makefile
make start-backend
```

### Frontend (React)
```bash
cd web

# Install dependencies (uses Bun)
bun install

# Development server (Vite)
bun run dev

# Build for production
bun run build

# Lint and format
bun run lint
bun run lint:fix

# i18n management
bun run i18n:extract    # Extract translation keys
bun run i18n:sync       # Sync translations
```

### Combined Build & Run
```bash
# Build frontend + run backend in one command
make all

# Docker
docker-compose up -d
```

### Testing & Single Test
```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./controller/

# Run a single test
go test ./controller/ -run TestChannelList

# Run with verbose output
go test ./... -v
```

## Architecture Overview

### Directory Structure - Key Layers

**HTTP Request Flow**: Router → Middleware → Controller → Service → Model/Relay → Database/External API

1. **`router/`** - Route registration and HTTP handlers setup
2. **`controller/`** (42 files) - HTTP request handlers, request validation, response formatting
3. **`relay/`** (18 files) - **Critical**: API format conversion and provider routing
   - `relay/channel/` - Provider-specific implementations
   - `relay/common_handler/` - Shared logic (prompt processing, token counting)
   - Format adapters: `claude_handler.go`, `gemini_handler.go`, etc.
4. **`service/`** - Business logic (payments, token encoding, HTTP clients)
5. **`model/`** - Data models and database CRUD operations
6. **`middleware/`** - Auth, logging, CORS, request/response middleware

### Data Models & Database

Key models in `model/`:
- **Channel** - API provider configurations (API key, base URL, model mappings)
- **Model** - Model metadata and pricing (id, owner, pricing_per_1k_tokens)
- **User** - User accounts, quotas, roles
- **Token** - User authentication tokens
- **Billing** - Usage records for billing

Database support: SQLite (default), MySQL, PostgreSQL via GORM

### Relay & API Compatibility

The `relay/` directory handles the complex task of converting between different provider APIs:

- **Input**: Standardized OpenAI-compatible format
- **Processing**: Channel model mapping, prompt preprocessing, token counting
- **Output**: Provider-specific format (Claude, Gemini, custom, etc.)
- **Special handling**: Streaming responses, images, embeddings, audio

Example: OpenAI chat request → detect target model → route to correct relay handler → convert format → call upstream → parse response → convert back to OpenAI format

### Frontend Structure (web/src/)

```
pages/       - Route-based page components
components/  - Reusable UI components (Semi Design)
services/    - API client (axios-based)
context/     - React Context for global state
hooks/       - Custom React hooks
i18n/        - Multi-language configuration (i18next)
```

Frontend uses **Semi Design** UI components (Alibaba's design system), **Tailwind CSS**, and **i18next** for i18n.

## Environment Configuration (.env)

Critical for development/deployment:
```
PORT=3000                           # Server port
SQL_DSN=                            # Database connection string
REDIS_CONN_STRING=                  # Redis for caching
SESSION_SECRET=your-secret          # ⚠️ REQUIRED for multi-instance deployments
CRYPTO_SECRET=your-secret           # ⚠️ REQUIRED for Redis encryption
DEBUG=true                          # Enable debug logging
GIN_MODE=debug                      # Gin framework mode
RELAY_TIMEOUT=300                   # API request timeout (seconds)
STREAMING_TIMEOUT=300               # Stream response timeout
CHANNEL_UPDATE_FREQUENCY=1800       # Channel cache update interval
```

For multi-machine deployments: must set `SESSION_SECRET` (ensures login consistency) and `CRYPTO_SECRET` (encrypts Redis data).

Copy `.env.example` to `.env` and configure before running.

## Common Development Tasks

### Adding a New Channel/Provider
1. Add channel config to database via UI or migration
2. Create handler in `relay/channel/[provider]/` if needed
3. Add format conversion in appropriate relay handler
4. Update model mappings in channel configuration
5. Test via `/v1/chat/completions` endpoint with channel specified

### Adding a New Model
1. Add model record to database (typically via admin panel)
2. Create model-to-channel mapping
3. Set pricing per 1k tokens for billing
4. Test with actual API calls

### Debugging API Relay Issues
- Check `relay/relay_adaptor.go` for routing logic
- Review request/response logs in `logger/`
- Use `DEBUG=true` in .env for verbose output
- Inspect channel configuration (API key, base URL, model mappings)

### Frontend Development
- Components in `web/src/components/`
- Pages in `web/src/pages/`
- Services (API calls) in `web/src/services/`
- Add i18n keys to language files, then run `bun run i18n:extract`

## Linting & Code Quality

**Backend**: No automatic linter configured; use `go fmt` manually
```bash
gofmt -w .
go vet ./...
```

**Frontend**: ESLint + Prettier configured
```bash
cd web
bun run eslint:fix      # Fix ESLint issues
bun run lint:fix        # Format with Prettier
```

## Docker Deployment

- **Dockerfile**: Multi-stage build (Bun frontend + Go binary)
- **docker-compose.yml**: Complete stack with optional PostgreSQL/Redis services
- Working directory: `/data` (mount for persistence)
- Exposed port: 3000

## Key Implementation Details

### Authentication & Middleware
- JWT tokens and session-based auth supported
- Multiple login providers: GitHub, LinuxDO, Telegram, OIDC, local
- Middleware validation in `middleware/` directory

### Payment & Billing
- Per-model token-based pricing
- Billing records stored in database
- Integration with Stripe and alternative payment gateways

### WebSocket Support
- Real-time chat/streaming via `relay/websocket.go`
- Long-running tasks (Midjourney, Suno) support

### Caching Strategy
- Redis optional but recommended for production
- Channel configurations cached
- User quota caching

## Testing & Debugging

- **Test location**: Typically `*_test.go` files in same directory as code
- **Test pattern**: Use `-run TestName` flag for single tests
- **Debug mode**: Set `DEBUG=true` and `GIN_MODE=debug` in .env
- **Logs**: Check application logs for relay and authentication errors

## Important Project Patterns

1. **Error Handling**: Controller handlers return `(interface{}, error)` - let middleware handle HTTP response
2. **Database Transactions**: Use GORM's `.WithContext()` for proper transaction handling
3. **Relay Format Conversion**: Always validate both request and response formats match target provider
4. **Token Counting**: Critical for accurate billing - implemented in `relay/common_handler/`
5. **Channel Selection**: Weighted random algorithm in channel configuration

## Deployment Notes

- **Single Instance**: SQLite + local cache sufficient
- **Multi-Instance**: Requires
  - Remote database (MySQL/PostgreSQL)
  - Redis for shared caching
  - `SESSION_SECRET` + `CRYPTO_SECRET` environment variables
  - Same binary version across instances

## Documentation & References

- README.md - High-level overview and deployment guides
- .env.example - All configurable environment variables
- GitHub workflows in `.github/workflows/` - CI/CD pipeline reference
- One API upstream project - Original architecture inspiration

