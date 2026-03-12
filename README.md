# Sea Battle Server

WebSocket game server for the Sea Battle multiplayer game built in Go.

## Tech Stack

- **Go 1.22**
- **[coder/websocket](https://github.com/coder/websocket) v1.8+** — WebSocket transport
- **[chi](https://github.com/go-chi/chi)** — HTTP router
- **slog** — structured logging (stdlib)

## Architecture

Clean architecture with four layers, each with a single responsibility:

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Domain | `internal/domain` | Game logic, rules, entities — zero external deps |
| Repository | `internal/repository` | In-memory storage with thread-safe access |
| Service | `internal/service` | Room management, game orchestration |
| Transport | `internal/transport` | WebSocket handler, HTTP routes, middleware |

## Getting Started

### Prerequisites

- Go 1.22+
- Docker (optional)

### Run locally

```bash
make run
```

### Run with Docker

```bash
docker-compose up
```

### Run tests

```bash
make test
```

### Coverage report

```bash
make cover
```

### Build binary

```bash
make build
# output: bin/sea-battle-server
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server listen port |
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `TURN_TIMEOUT` | `30s` | Max time per player turn |
| `RECONNECT_GRACE` | `60s` | Grace period for player reconnection |
| `ROOM_CLEANUP_INTERVAL` | `60s` | How often stale rooms are purged |
| `MAX_ROOMS` | `1000` | Maximum concurrent game rooms |

## API

### WebSocket

```
ws://localhost:8080/ws?game={CODE}&player={PLAYER_ID}
```

Connect to an existing game room. Messages are JSON-encoded.

### REST

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/games` | Create a new game room |
| `GET` | `/api/games/{code}` | Get game status by code |

## Docker

Multi-stage build (`golang:1.22-alpine` → `alpine:3.19`). Final image is under 20 MB. Health check polls `/health` every 30 s.

```bash
# Build image
docker build -t sea-battle-server .

# Run container
docker run -p 8080:8080 sea-battle-server
```

Resource limits when using Compose: 1 CPU / 256 MB max, 0.25 CPU / 64 MB reserved.

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go               # Entry point
└── internal/
    ├── config/
    │   └── config.go             # Env-based configuration
    ├── domain/
    │   ├── board.go              # Board logic
    │   ├── cell.go               # Cell state
    │   ├── errors.go             # Domain errors
    │   ├── game.go               # Game state machine
    │   ├── player.go             # Player model
    │   ├── point.go              # Coordinate type
    │   └── ship.go               # Ship placement & hit detection
    ├── repository/
    │   ├── interfaces.go         # Repository contracts
    │   └── memory.go             # In-memory implementation
    ├── service/
    │   ├── game_service.go       # Game lifecycle & turn logic
    │   └── room_manager.go       # Room creation & cleanup
    └── transport/
        ├── handler.go            # WebSocket connection handler
        ├── messages.go           # WS message types (in/out)
        ├── middleware.go         # HTTP middleware (logging, recovery)
        └── router.go             # Chi router setup
```

## License

MIT
