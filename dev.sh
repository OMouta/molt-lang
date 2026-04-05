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
    ;;
  *)
    echo "usage: ./dev.sh {build|test|lint|format}" >&2
    exit 1
    ;;
esac
