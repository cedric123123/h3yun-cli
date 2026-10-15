param([string]$GoCommand = 'go')
$ErrorActionPreference = 'Stop'
$projectDir = Split-Path -Parent $PSScriptRoot
$previousOS = $env:GOOS
$previousArch = $env:GOARCH
$previousCGO = $env:CGO_ENABLED
Push-Location $projectDir
try {
    $env:CGO_ENABLED = '0'
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    & $GoCommand test ./... -cover
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
    & $GoCommand vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed' }
    New-Item -ItemType Directory -Force -Path 'dist' | Out-Null
    $targets = @(
        @{ OS = 'windows'; Arch = 'amd64'; Name = 'h3yun.exe' },
        @{ OS = 'linux'; Arch = 'amd64'; Name = 'h3yun-linux-amd64' },
        @{ OS = 'darwin'; Arch = 'arm64'; Name = 'h3yun-darwin-arm64' },
        @{ OS = 'darwin'; Arch = 'amd64'; Name = 'h3yun-darwin-amd64' }
    )
    foreach ($target in $targets) {
        $env:GOOS = $target.OS
        $env:GOARCH = $target.Arch
        & $GoCommand build -trimpath -ldflags '-s -w' -o (Join-Path 'dist' $target.Name) ./cmd/h3yun
        if ($LASTEXITCODE -ne 0) { throw ('Build failed for ' + $target.OS + '/' + $target.Arch) }
    }
    Get-FileHash -Algorithm SHA256 -Path 'dist/h3yun.exe', 'dist/h3yun-linux-amd64', 'dist/h3yun-darwin-arm64', 'dist/h3yun-darwin-amd64'
} finally {
    $env:GOOS = $previousOS
    $env:GOARCH = $previousArch
    $env:CGO_ENABLED = $previousCGO
    Pop-Location
}
