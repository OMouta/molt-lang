param(
  [Parameter(Mandatory = $true)]
  [ValidateSet("build", "test", "lint", "format", "docs", "docs:build", "docs:gen")]
  [string]$Task
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path

Push-Location $root
try {
  switch ($Task) {
    "build"      { go build ./... }
    "test"       { go test ./... }
    "lint"       { go vet ./... }
    "format"     { gofmt -w . }
    "docs:gen"   { go run ./cmd/docgen }
    "docs"       {
      go run ./cmd/docgen
      Push-Location docs
      npm run docs:dev
      Pop-Location
    }
    "docs:build" {
      go run ./cmd/docgen
      Push-Location docs
      npm run docs:build
      Pop-Location
    }
  }

  if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
  }
} finally {
  Pop-Location
}
