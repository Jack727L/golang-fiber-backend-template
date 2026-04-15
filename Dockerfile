FROM golang:1.26.2 AS builder

WORKDIR /app

# Copy module files for both root and env sub-module
COPY go.mod go.sum ./
COPY env ./env

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o app main.go

# ─── runtime image ────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

WORKDIR /root/

COPY --from=builder /app/app .

EXPOSE 3000

CMD ["./app"]
