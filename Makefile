# Local binary name (replace YOUR_BINARY_NAME when you fork).
BINARY_NAME ?= YOUR_BINARY_NAME

build:
	go build -o /tmp/$(BINARY_NAME) main.go

run: build
	/tmp/$(BINARY_NAME)

watch:
	reflex -s -r '\.go$$' make run

test:
	./tools/runTests.sh

test-verbose:
	./tools/runTests.sh -v

docs:
	./tools/generateDocs.sh

sqlc:
	./tools/generateSQLC.sh

.PHONY: build run watch test test-verbose docs sqlc
