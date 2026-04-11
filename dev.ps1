param(
  [Parameter(Mandatory = $true)]
  [ValidateSet("build", "test", "lint", "format", "format:check", "docs", "docs:build", "docs:gen")]
  [string]$Task
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path

Push-Location $root
try {
  switch ($Task) {
    "build"      { go build ./... }
    "test"       { go test ./... }
    "lint"       { go vet ./... }
    "format"     {
      gofmt -w .
      if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
      }

      go run ./cmd/molt fmt .
    }
    "format:check" {
      $unformattedGo = gofmt -l .
      if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
      }

      if ($unformattedGo) {
        $unformattedGo | Write-Output
        exit 1
      }

      go run ./cmd/molt fmt --check .
    }
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
