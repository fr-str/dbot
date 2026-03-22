# AGENTS.md - Developer Guide for dbot

dbot is a Discord bot written in Go (v1.25) that plays music, manages playlists, and provides audio/video features. Uses SQLite, sqlc for DB code generation, and goose for migrations.

## Build Commands

```bash
# Run with hot reload (uses .air.toml)
air

# Build manually (debug tags)
go build -o ./tmp/main -tags 'debug' ./cmd/dbot/main.go

# Build for production
go build -tags 'production' ./cmd/dbot/main.go

# Tidy dependencies
go mod tidy
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests in specific package with verbose output
go test -v ./pkg/player

# Run specific test function (supports regex)
go test -v -run TestPlaylist ./pkg/player
go test -v -run "TestParseTime/success" ./pkg/ffmpeg

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...
```

## Code Generation

```bash
sqlc generate          # Generate database code
goose ... up           # Run migrations
```

## Linting & Type Checking

```bash
go vet ./...           # Vet
go fmt ./...           # Format
```

## Code Style

### Imports (3 groups, blank line between)
1. Standard library
2. External packages (github.com, etc.)
3. Internal packages (dbot/...)

```go
import (
    "context"
    "fmt"
    "os"

    "github.com/bwmarrin/discordgo"
    "github.com/fr-str/log"

    "dbot/pkg/config"
    "dbot/pkg/store"
)
```

### Naming Conventions
- PascalCase for exported, camelCase for private
- Use descriptive names; avoid abbreviations (except `Ctx` for `Context`)
- Prefix interfaces with `er` (e.g., `Reader`, `Player`)
- Test functions: `Test<Unit>_<scenario>`

### Error Handling
- Return errors, don't panic (except init failures)
- Use custom error wrappers with context:
```go
func dbotErr(msg string, vars ...any) error {
    return fmt.Errorf("dbot: "+msg+": ", vars...)
}
```
- Use `errors.Wrap` or `%w` for chaining; log errors before returning

### Types & Interfaces
- Use concrete types; define interfaces only when needed
- Embed types for composition (e.g., `*discordgo.Session` in `DBot`)
- Use `context.Context` as first parameter for cancellable functions
- Prefer values over pointers; use pointers only when needed

### Testing
- Prefer `github.com/stretchr/testify` for assertions:
```go
require.NoError(t, err)    // stops test on failure
assert.Equal(t, expected, actual)  // continues on failure
```
- Test files: `*_test.go` in same package
- Use table-driven tests with subtests:
```go
func TestParseTime(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        t.Run("parses seconds", func(t *testing.T) { /* test */ })
    })
    t.Run("fail", func(t *testing.T) { /* test */ })
}
```

## Database
- Generated with sqlc - don't edit manually
- SQL queries in `db/queries/*.sql`, migrations in `db/migrations/`
- Use WAL mode + PRAGMA synchronous=NORMAL

## Logging
- Use `github.com/fr-str/log`: `log.Error`, `log.Warn`, `log.Info`, `log.Trace`
- Add context: `log.Any`, `log.String`, etc.

## Configuration
- Use `github.com/fr-str/env`; define in `pkg/config/config.go`

## Project Structure
```
cmd/           - Entry points
pkg/           - Core packages
  bot/         - Discord bot logic, handlers, commands
  player/      - Music player
  store/       - DB queries (generated)
  ffmpeg/      - Audio/video processing
db/            - SQL files and migrations
```

## Common Patterns

### Start bot:
```go
config.Load()
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()
db, err := db.ConnectStore(ctx, "./test.db", "")
d := dbot.Start(ctx, dg, db)
```

### Add slash command (pkg/bot/commands.go):
```go
{
    Name: "command-name",
    Description: "Description",
    Options: []*discordgo.ApplicationCommandOption{
        { Type: discordgo.ApplicationCommandOptionString, Name: "option", Description: "..." },
    },
},
```

### FFmpeg clip:
```go
clip := ffmpeg.Clip{Start: 10 * time.Second, End: 30 * time.Second}
f, err := ffmpeg.ToDiscordMP4(ctx, file, false, clip)
```

## Environment Variables
| Variable | Required | Description |
|----------|----------|-------------|
| `TOKEN` | Yes | Discord bot token |
| `GUILD_ID` | Yes | Discord guild ID |
| `ENV` | No | Environment (dev/prod) |
| `DB_DIR` | No | Database directory |
| `TMP_PATH` | No | Temporary file path |
| `FFMPEG_HW_ACCEL` | No | FFmpeg hardware acceleration |

## Important Notes
- SQLite WAL mode - beware concurrent writes
- Player has two loops (musicLoop, soundLoop) - understand before modifying
- Voice connections need proper cleanup - always call Close()
- Some packages have debug build tags - use `-tags 'debug'` locally
