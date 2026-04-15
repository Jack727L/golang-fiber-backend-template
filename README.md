<div align="center">

# YOUR_REPO_NAME

**REST API starter** — [Fiber](https://gofiber.io/), PostgreSQL + [sqlc](https://sqlc.dev/), [Sqitch](https://sqitch.org/), Redis jobs, Swagger, [testcontainers](https://testcontainers.com/).

<br/>

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![Fiber](https://img.shields.io/badge/Fiber-v2-6366f1?style=for-the-badge)](https://gofiber.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis&logoColor=white)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://docs.docker.com/compose/)

<br/><br/>

[Features](#features) · [Quick start](#quick-start) · [Makefile](#makefile) · [Tools](#tools) · [API](#api-overview) · [Testing](#testing) · [Docker](#docker-image) · [AWS](#deploying-on-aws) · [Layout](#project-layout)

</div>

<br/>

---

## Features

| Area | What you get |
| :--- | :--- |
| **HTTP** | Fiber, CORS, logging, validation ([validator](https://github.com/go-playground/validator)) |
| **Auth** | Register, login, refresh, logout under `/auth` |
| **Users** | Bearer-protected `/users/me` profile routes |
| **Data** | `pgx` pool · SQL in `db/queries/` · generated code in `db/sqlc/` |
| **Jobs** | Redis + executor in `core/jobs/` |
| **Ops** | `/healthz`, `/healthcheck`, `/readyz` (readiness checks DB) |
| **Docs** | Swagger annotations → `docs/` |

---

## Requirements

| Need | Notes |
| :--- | :--- |
| **Go** | 1.26+ (`go.mod`; latest patch recommended) |
| **Docker** | Local Postgres/Redis via Compose · testcontainers for tests |
| **Sqitch** | Run migrations from `db/sqitch` before first boot |
| **Optional** | `sqlc`, `swag`, `reflex` (`make watch`) |

---

## Quick start

### 1 · Environment

```bash
cp .env.example .env
```

Edit `.env` if ports or credentials differ from the defaults.

### 2 · Postgres & Redis

```bash
docker compose up -d
```

Defaults: Postgres **5432**, Redis **6379**.

### 3 · Migrations

From the repository root, use a URI that matches `.env`:

```bash
cd db/sqitch
sqitch deploy "db:pg://appuser:apppassword@localhost:5432/appdb"
cd ../..
```

### 4 · Run the API

```bash
go run .
```

Listens on **`:3000`**. Set `BACKEND_ENV` in `.env` (for example `local-dev`).

---

## Makefile

| Target | What it does |
| :--- | :--- |
| `make build` | Binary → `/tmp/$(BINARY_NAME)` (default **`YOUR_BINARY_NAME`** in `Makefile`) |
| `make run` | Build + run |
| `make watch` | Rebuild on `.go` changes · needs [`reflex`](https://github.com/cespare/reflex) |
| `make test` | `./tools/runTests.sh` (Docker required) |
| `make test-verbose` | Same with `-v` |
| `make docs` | Swagger via `./tools/generateDocs.sh` · needs `swag` |
| `make sqlc` | Regenerate `db/sqlc/` · needs `sqlc` |

---

## Tools

Scripts live in [`tools/`](tools/). Run them from the **repository root**.

| Script | Role | Makefile |
| :--- | :--- | :--- |
| [`generateDocs.sh`](tools/generateDocs.sh) | Swagger / OpenAPI from handler comments | `make docs` |
| [`generateSQLC.sh`](tools/generateSQLC.sh) | Typed Go DB code from SQL | `make sqlc` |
| [`runTests.sh`](tools/runTests.sh) | Integration tests (Docker) | `make test` · `make test-verbose` |

### `generateDocs.sh`

Regenerates **`docs/`** (e.g. `swagger.json`, `swagger.yaml`, `docs.go`) from Swag annotations in Go source. Run after you change route handlers or doc comments.

```bash
./tools/generateDocs.sh
```

**Requires** [swag](https://github.com/swaggo/swag):

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

**Runs:** `swag init --parseDependency --parseInternal`

---

### `generateSQLC.sh`

Regenerates **`db/sqlc/`** from `sqlc.yaml`, SQL in **`db/queries/`**, and schema files under **`db/sqitch/deploy/`**. Run after any change to queries or deploy SQL.

```bash
./tools/generateSQLC.sh
```

**Requires** [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) on your `PATH`.

**Runs:** `sqlc generate` (project root; uses `sqlc.yaml`).

---

### `runTests.sh`

Runs **`go test ./tests/...`** with `BACKEND_ENV=test` and the right Docker wiring. **Do not** call `go test` directly for these suites—the script sets up containers and env.

```bash
./tools/runTests.sh                    # default: testcontainers (Postgres per run)
./tools/runTests.sh -v                 # verbose test output
./tools/runTests.sh -t TestUserTestSuite   # -run filter (specific suite / test)
./tools/runTests.sh --async            # Compose stack + Redis (see tests/testing.docker-compose.yml)
./tools/runTests.sh --async --temp     # same, then tear down Compose volumes
```

**Requires:** Docker daemon. **`--async`** also needs **Sqitch** (runs `sqitch deploy` in `db/sqitch`) and uses `tests/testing.docker-compose.yml`.

Extra `go test` flags can be passed through (see script for full parsing).

---

### Live reload (`make watch`)

Not a script under `tools/`—**`make watch`** uses [`reflex`](https://github.com/cespare/reflex) to rebuild when `.go` files change. Install reflex, then:

```bash
make watch
```

---

## API overview

| Area | Routes |
| :--- | :--- |
| **Auth** | `POST /auth/register` · `POST /auth/login` · `POST /auth/refresh` · `POST /auth/logout` |
| **Users** | `GET` · `PUT` · `DELETE` `/users/me` (Bearer) |
| **Health** | `GET /healthz` · `GET /healthcheck` · `GET /readyz` |

Regenerate OpenAPI under `docs/` after changing handler annotations.

---

## Testing

Use **`tools/runTests.sh`** — see the [Tools](#tools) section for every flag and prerequisite. Short version:

```bash
./tools/runTests.sh                # default
./tools/runTests.sh -v
./tools/runTests.sh -t TestName
./tools/runTests.sh --async
```

Requires a running **Docker** daemon; **`--async`** needs **Sqitch** as well.

---

## Docker image

Multi-stage build, distroless runtime:

```bash
docker build -t YOUR_IMAGE_NAME .
```

Uncomment the sample `app` service in `docker-compose.yml` to run the API in Compose.

---

## Deploying on AWS

This template does **not** ship Terraform, CloudFormation, or CDK. Below is how the app’s **environment variables** map to typical AWS managed services and what to wire in your own IaC or console setup.

### RDS for PostgreSQL

| Setting | AWS notes |
| :--- | :--- |
| `POSTGRES_HOST` | RDS **endpoint** (writer hostname), same VPC as the app or reachable privately. |
| `POSTGRES_PORT` | Usually `5432`. |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` | Master user or an app user you create in the DB; store secrets in **Secrets Manager** or **SSM Parameter Store**, not in the image. |
| `POSTGRES_DATABASE` | Your database name on the instance. |
| `POSTGRES_SSLMODE` | Use `require` or `verify-full` in production (RDS supports TLS). The app defaults to **`require`** when this is unset. |

**Networking:** RDS security group should allow inbound **5432** only from the app’s security group (or a shared private subnet pattern), not from `0.0.0.0/0`.

**Migrations:** Apply Sqitch from a **CI job**, **bastion**, or **one-off task** that can reach RDS (same idea as local `sqitch deploy`, with the RDS URI). The running API container does not apply migrations for you.

### ElastiCache for Redis

| Setting | AWS notes |
| :--- | :--- |
| `REDIS_HOST` | Primary **configuration endpoint** (cluster mode) or **primary endpoint** (non-cluster), per your ElastiCache design. |
| `REDIS_PORT` | Often `6379` (or the port shown in the console). |
| `REDIS_PASSWORD` | Auth token / password if **AUTH** is enabled. |
| `REDIS_TLS_ENABLED` | Set to `true` when using **encryption in transit** (common on ElastiCache Serverless and many TLS-enabled clusters). |

**Networking:** Security group for Redis should allow the app SG on the Redis port only.

### Application runtime (typical patterns)

- **Amazon ECS on Fargate** (or EC2): Build and push the image from this repo’s `Dockerfile` to **ECR**; define task env vars from Secrets Manager / SSM; place tasks behind an **Application Load Balancer** whose target group forwards to **container port 3000**. Set `BACKEND_ENV` to `staging` or `prod` as you define in `env/env.go`.
- **EC2 / Auto Scaling:** Run the binary or Docker with the same env vars; use **Systems Manager** or **user data** only for non-secret bootstrap—inject DB/Redis secrets via the agent or instance role + Secrets Manager.
- **App Runner / Lambda:** Possible with adaptation (Lambda would need API Gateway and often a different packaging model than this long-lived server).

### Health checks for load balancers

The app exposes **`GET /healthz`** and **`GET /readyz`** (readiness includes a DB ping). Point ALB/ECS health checks at **`/healthz`** for liveness or **`/readyz`** if you want the target removed when Postgres is unreachable.

---

## Project layout

```
├── core/           # Errors, jobs, Redis
├── db/
│   ├── queries/    # SQL (sqlc input)
│   ├── sqlc/       # Generated — do not edit by hand
│   └── sqitch/     # Migrations
├── docs/           # Swagger (generated)
├── env/            # Env helpers (submodule)
├── middlewares/
├── routers/
├── services/
├── setup/          # App wiring & health routes
├── tests/
├── tools/          # generateDocs.sh, generateSQLC.sh, runTests.sh
├── main.go
├── Dockerfile
└── docker-compose.yml
```

---

## Customizing

**Go module (required for a fork)** — replace **`github.com/yourusername/go-api-starter`** in `go.mod`, `env/go.mod`, every Go import, and the **`%uri`** line in `db/sqitch/sqitch.plan`. Regenerate Swagger if you change doc comments in `main.go` (`make docs`). Then:

```bash
go mod tidy
```

**Other placeholders (optional, for naming)** — search and replace to taste:

| Placeholder | Where |
| :--- | :--- |
| `YOUR_BINARY_NAME` | `Makefile` (`BINARY_NAME`) · default output `/tmp/YOUR_BINARY_NAME` |
| `YOUR_IMAGE_NAME` | README Docker example · `docker build -t …` |
| `YOUR_REPO_NAME` | README title |
| `YOUR_SQITCH_PROJECT` | `db/sqitch/sqitch.plan` (`%project`) · first line of each file under `db/sqitch/deploy/`, `revert/`, `verify/` (must stay consistent) |
| `YOUR_NAME` / `your.email@example.com` | Author lines in `db/sqitch/sqitch.plan` (new migrations should use your identity) |

---

## License

Add a `LICENSE` file when you ship or fork this template.
