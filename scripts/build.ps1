$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path bin | Out-Null

go build `
  -trimpath `
  -o bin/lab.exe `
  ./cmd/lab

Write-Host ""
Write-Host "Built: bin/lab.exe"

.\bin\lab.exe version