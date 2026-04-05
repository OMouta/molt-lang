param(
  [Parameter(Mandatory = $true)]
  [ValidateSet("build", "test", "lint", "format")]
  [string]$Task
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Path

Push-Location $root
try {
  switch ($Task) {
    "build" { go build ./... }
    "test" { go test ./... }
    "lint" { go vet ./... }
    "format" { gofmt -w . }
  }

  if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
  }
} finally {
  Pop-Location
}
