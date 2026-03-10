# Promptman

> A CLI-first HTTP API client with a local daemon — built for developers who live in the terminal.

[![Go](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](#license)

Promptman lets you define HTTP requests as YAML collections, run them from the CLI, test responses with JavaScript, and manage environments — all without a GUI.

```bash
# Run a single request
promptman run users/health

# Run an entire collection
promptman run --collection users --env staging

# Check daemon status
promptman status
```

---

## Features

- **YAML-first collections** — define requests, headers, and bodies in version-controlled files
- **Daemon architecture** — a lightweight local server handles execution; the CLI is a thin client
- **JavaScript test scripts** — Postman-compatible `pm.*` API with `pm.test`, `pm.expect`, lifecycle hooks
- **Environment management** — named environments with variable interpolation, create/read/update/delete via CLI
- **Auto-start** — daemon starts automatically when you run your first request in a session
- **Multiple output formats** — `json`, `table`, `minimal`; machine-readable by default
- **WebSocket hub** — real-time event broadcasting for test reporters and IDE integrations

---

## Installation

### From source

```bash
git clone https://github.com/khanhnguyen/promptman.git
cd promptman

# Build both binaries
make build-all

# Move to your PATH
cp bin/promptman /usr/local/bin/promptman
```

> **Requirements:** Go 1.21+

---

## Quick Start

### 1. Initialize a project

```bash
mkdir my-api && cd my-api
promptman init
```

This creates `.promptman/` with the project structure:

```
.promptman/
├── collections/   # YAML request collections
├── environments/  # Named environment files
└── tests/         # JavaScript test scripts
```

### 2. Create a collection

`.promptman/collections/users.yaml`:

```yaml
name: Users API
baseUrl: https://api.example.com
defaults:
  headers:
    Content-Type: application/json
    Authorization: Bearer {{token}}

folders:
  - id: auth
    name: Authentication
    requests:
      - id: health
        method: GET
        path: /health

      - id: login
        method: POST
        path: /auth/login
        body:
          type: json
          content:
            email: "{{email}}"
            password: "{{password}}"

  - id: profile
    name: User Profile
    requests:
      - id: me
        method: GET
        path: /users/me
```

### 3. Create an environment

`.promptman/environments/dev.yaml`:

```yaml
name: dev
variables:
  token: "my-dev-token"
  email: "dev@example.com"
  password: "secret"
```

### 4. Run requests

```bash
# Single request (auto-starts daemon on first run)
promptman run users/auth/login --env dev

# Run a whole collection
promptman run --collection users --env dev

# Stop on first failure
promptman run --collection users --stop-on-error
```

---

## Collections

Collections are YAML files that group related HTTP requests. Each file lives in `.promptman/collections/`.

| Field | Description |
|-------|-------------|
| `name` | Display name |
| `baseUrl` | Base URL prepended to all `path` values |
| `defaults.headers` | Headers merged into every request |
| `folders` | Request groups with `id`, `name`, and `requests` |

**Request fields:**

| Field | Description |
|-------|-------------|
| `id` | Request identifier used in CLI paths (`collection/folder/id`) |
| `method` | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE` |
| `path` | Path appended to `baseUrl` |
| `headers` | Per-request header overrides |
| `body.type` | Body type: `json`, `form`, `raw` |
| `body.content` | Body content (object for `json`/`form`, string for `raw`) |

Variable interpolation: `{{variable_name}}` anywhere in `headers`, `path`, or `body`.

---

## Environments

Environments store named variable sets. They are YAML files in `.promptman/environments/`.

```yaml
name: staging
variables:
  base_url: "https://staging.api.example.com"
  token: "stg-abc123"
```

**CLI commands:**

```bash
# List environments
promptman env list

# Get a variable
promptman env get staging token

# Set a variable
promptman env set staging token "new-value"

# Delete a variable
promptman env unset staging token
```

---

## Test Scripts

Promptman includes a JavaScript testing engine (powered by [Goja](https://github.com/dop251/goja)) compatible with the Postman `pm.*` API.

Place test files in `.promptman/tests/<collection-id>.test.js`.

### Example

```javascript
module.exports = {
  // Lifecycle hooks
  beforeAll: function(pm) {
    pm.environment.set("created_id", "");
  },

  afterAll: function(pm) {
    pm.environment.unset("created_id");
  },

  beforeEach: function(pm) {
    pm.variables.set("ok", "false");
  },

  // Test specific request: "posts/create-post"
  "posts/create-post": function(pm) {
    pm.test("Returns 201 Created", function() {
      pm.expect(pm.response.status).to.equal(201);
    });

    var post = pm.response.json();

    pm.test("Has an ID", function() {
      pm.expect(post.id).to.be.a("number");
    });

    // Share data across tests via environment
    pm.environment.set("created_id", String(post.id));
    pm.variables.set("ok", "true");
  },

  // Wildcard: matches posts/get-post, posts/update-post, etc.
  "posts/*": function(pm) {
    pm.test("Post endpoints return success", function() {
      pm.expect(pm.response.status).to.be.below(300);
    });
  },

  // Glob: matches any depth under comments/
  "comments/**": function(pm) {
    pm.test("Comments endpoint returns 200", function() {
      pm.expect(pm.response.status).to.equal(200);
    });
  },
};
```

### `pm.*` API

| API | Description |
|-----|-------------|
| `pm.response.status` | HTTP status code |
| `pm.response.json()` | Parsed JSON body |
| `pm.response.text()` | Raw body string |
| `pm.response.headers` | Response headers object |
| `pm.test(name, fn)` | Register a named test assertion |
| `pm.expect(value)` | Chai BDD assertion (`to.equal`, `to.be.a`, `to.have.property`, ...) |
| `pm.environment.get/set/unset` | Persistent variables (survive across requests) |
| `pm.variables.get/set/unset` | Local variables (scoped to current request) |

### Key matching

Test functions are matched to requests by their key:

| Pattern | Matches |
|---------|---------|
| `"posts/create-post"` | Exact request ID |
| `"posts/*"` | Any single segment under `posts/` |
| `"comments/**"` | Any depth under `comments/` |

---

## CLI Reference

```
promptman [flags] <command>

Commands:
  init      Initialize a new promptman project
  run       Execute HTTP requests via the daemon
  env       Manage environment variables
  status    Show daemon status
  version   Show version info

Global flags:
  --project-dir string   Project directory (default: cwd)
  --format string        Output format: json, table, minimal (default: json)
  --env string           Environment override (run command)
  --yes                  Skip confirmation prompts
  --dry-run              Print without executing
```

---

## Architecture

Promptman uses a **CLI + local daemon** architecture:

```
┌─────────────┐   HTTP/JSON   ┌─────────────────────────────────────┐
│  promptman  │ ────────────► │  promptman-daemon (local HTTP server)│
│  (CLI)      │               │                                      │
└─────────────┘               │  ┌──────────┐  ┌────────────────┐   │
                              │  │ Request  │  │   Environment  │   │
                              │  │ Engine   │  │   Service      │   │
                              │  └──────────┘  └────────────────┘   │
                              │  ┌──────────┐  ┌────────────────┐   │
                              │  │Collection│  │   WebSocket    │   │
                              │  │ Service  │  │   Hub          │   │
                              │  └──────────┘  └────────────────┘   │
                              └─────────────────────────────────────┘
```

The daemon starts automatically when you run your first command and persists across sessions via a `.promptman/.daemon.lock` file. You can also control it manually:

```bash
# Start daemon manually
promptman daemon start --project-dir .

# Check status
promptman status
```

---

## Development

```bash
# Run tests
make test

# Run with race detector
make test-race

# Build CLI only
make build-cli

# Build daemon only
make build-daemon

# Build both
make build-all

# Lint
make lint

# Format
make fmt
```

**Project layout:**

```
cmd/
  cli/          # CLI entry point
  daemon/       # Daemon entry point
internal/
  cli/          # Cobra commands, formatter, daemon client
  collection/   # YAML collection loading
  daemon/       # Manager, server, registrars, lock file
  environment/  # Environment YAML service
  request/      # HTTP execution engine
  testing/      # JS sandbox, pm.* API, test runner, reporter
  ws/           # WebSocket hub
pkg/
  envelope/     # Consistent API response wrapper
  fsutil/       # Project directory helpers
  variable/     # Variable interpolation engine
```

---

## Contributing

1. Fork → branch (`feat/your-feature`) → PR
2. All PRs must pass `make test` and `make lint`
3. Commit messages follow [Conventional Commits](https://www.conventionalcommits.org): `feat:`, `fix:`, `refactor:`, `docs:`

---

## License

MIT — see [LICENSE](LICENSE).
