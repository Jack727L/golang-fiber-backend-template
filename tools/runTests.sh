#!/usr/bin/env bash
# Run tests using testcontainers (sync mode, default) or docker-compose (async mode).
#
# Usage:
#   ./tools/runTests.sh                         # sync mode (fast, isolated per suite)
#   ./tools/runTests.sh -t TestUserTestSuite    # run a specific test
#   ./tools/runTests.sh -v                      # verbose output
#   ./tools/runTests.sh --async                 # async mode (requires Docker Compose + Redis)
#   ./tools/runTests.sh --temp                  # tear down docker-compose after run
#
# Requirements: Docker daemon must be running.
# NEVER use `go test` directly — Docker setup is handled here.

set -e

ASYNC_MODE=false
TEMP_MODE=false
VERBOSE=""
TEST_PATTERN=""
EXTRA_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --async)     ASYNC_MODE=true; shift ;;
        --temp)      TEMP_MODE=true; shift ;;
        -v)          VERBOSE="-v"; shift ;;
        -t)          TEST_PATTERN="-run $2"; shift 2 ;;
        *)           EXTRA_ARGS+=("$1"); shift ;;
    esac
done

export BACKEND_ENV=test

if [ "$ASYNC_MODE" = true ]; then
    export ASYNC_MODE=true
    export USE_DOCKER_COMPOSE=true

    echo "Starting docker-compose services …"
    docker compose -f tests/testing.docker-compose.yml up -d --build

    # Wait for postgres
    echo "Waiting for Postgres …"
    until docker exec app_test_postgres pg_isready -U testuser -d testdb 2>/dev/null; do
        sleep 1
    done

    echo "Running migrations …"
    cd db/sqitch && sqitch deploy && cd ../..
else
    export ASYNC_MODE=false
    export USE_DOCKER_COMPOSE=false
fi

echo ""
echo "Running tests …"
echo "─────────────────────────────────────────────────"

go test $VERBOSE $TEST_PATTERN "${EXTRA_ARGS[@]}" ./tests/... -count=1 -timeout 300s

EXIT_CODE=$?

if [ "$ASYNC_MODE" = true ] && [ "$TEMP_MODE" = true ]; then
    echo ""
    echo "Tearing down docker-compose …"
    docker compose -f tests/testing.docker-compose.yml down -v
fi

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo "✓ All tests passed"
else
    echo "✗ Tests failed"
fi

exit $EXIT_CODE
