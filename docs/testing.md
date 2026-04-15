# Testing: local and CI/CD

Integration tests live under **`tests/`** and expect **Docker** (for [testcontainers](https://testcontainers.com/) or Compose). The entry point is always **`tools/runTests.sh`** — do not run `go test ./tests/...` by itself; the script sets `BACKEND_ENV=test` and any mode flags tests rely on.

---

## Local

### Prerequisites

| Requirement | Why |
| :--- | :--- |
| **Go** | Same version as `go.mod` (toolchain). |
| **Docker Desktop** (or Docker Engine + Compose) | Default mode starts Postgres via testcontainers; `--async` uses Compose. |

### Default (recommended): testcontainers

Starts an isolated Postgres container per run (no Sqitch on the host required for this path — migrations run from the test harness).

```bash
./tools/runTests.sh
```

Useful options:

```bash
./tools/runTests.sh -v                         # verbose
./tools/runTests.sh -t TestUserTestSuite       # single suite / pattern (passed to go test -run)
```

### Async mode: Compose + Sqitch

Exercises Redis and a shared Compose stack. Requires **Sqitch** on your `PATH` and uses **`tests/testing.docker-compose.yml`**.

```bash
./tools/runTests.sh --async
./tools/runTests.sh --async --temp            # tear down compose volumes after
```

### Troubleshooting (local)

- **“Cannot connect to Docker”** — start Docker Desktop / the daemon; testcontainers talks to the default socket.
- **Slow or flaky first run** — images are pulled on demand; subsequent runs are faster.
- **`--async` fails at `sqitch deploy`** — run from repo root; ensure Postgres in the test compose is healthy (`app_test_postgres`).

---

## CI/CD (GitHub Actions)

The repository includes **`.github/workflows/ci.yml`**, which runs the same default path as locally:

1. Check out the repo.
2. Install Go (version from `go.mod`).
3. Run **`bash ./tools/runTests.sh`** on `ubuntu-latest` (Docker is available on hosted runners).

Triggers: pushes and pull requests to **`main`** and **`master`** (adjust the workflow file if you use other branch names).

### After you fork or rename the module

Replace the placeholder module **`github.com/yourusername/go-api-starter`** in `go.mod`, imports, and `env/go.mod`, then commit — CI uses `go test` inside the script and needs a consistent module path.

### Enabling workflows

- **Your repo:** Actions are on by default for public repos; for private repos, enable **Actions** under **Settings → Actions → General**.
- **Forks:** Scheduled/workflow runs from forks may be restricted; open a **pull request** against the upstream repo to run CI there.

### Optional: async tests in CI

The default workflow only runs **sync** (testcontainers) mode — no Sqitch install, faster and simpler. To add **`--async`** in CI you would need extra steps (install Sqitch, run Compose, longer timeout). Prefer keeping async tests local or in a separate optional workflow if you need them in every run.

### Optional: other platforms

The same idea applies anywhere you have Docker and Go:

- **GitLab CI:** `image: golang` + `docker:dind` service, or a runner with Docker socket mounted.
- **Self-hosted:** Run `bash ./tools/runTests.sh` in a job; ensure Docker is available to the agent.

---

## Quick reference

| Context | Command |
| :--- | :--- |
| Local, default | `./tools/runTests.sh` |
| Local, verbose | `./tools/runTests.sh -v` |
| Local, async | `./tools/runTests.sh --async` |
| CI (this repo) | `bash ./tools/runTests.sh` (see workflow file) |
