#!/usr/bin/env sh
set -eu

task="${1:-}"

case "$task" in
  build)
    go build ./...
    ;;
  test)
    go test ./...
    ;;
  lint)
    go vet ./...
    ;;
  format)
    gofmt -w .
    go run ./cmd/molt fmt .
    ;;
  format:check)
    unformatted_go="$(gofmt -l .)"
    if [ -n "$unformatted_go" ]; then
      printf '%s\n' "$unformatted_go"
      exit 1
    fi
    go run ./cmd/molt fmt --check .
    ;;
  docs)
    go run ./cmd/docgen
    cd docs && npm run docs:dev
    ;;
  docs:build)
    go run ./cmd/docgen
    cd docs && npm run docs:build
    ;;
  docs:gen)
    go run ./cmd/docgen
    ;;
  *)
    echo "usage: ./dev.sh {build|test|lint|format|format:check|docs|docs:build|docs:gen}" >&2
    exit 1
    ;;
esac
