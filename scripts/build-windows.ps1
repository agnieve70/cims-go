param(
    [string]$Version = "dev",
    [string]$Addr = ":8090"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$distRoot = Join-Path $repoRoot "dist\windows"
$bundleRoot = Join-Path $distRoot "CIMS"
$exePath = Join-Path $bundleRoot "cims.exe"
$zipPath = Join-Path $distRoot ("CIMS-Windows-" + $Version + ".zip")
$issPath = Join-Path $repoRoot "packaging\windows\cims.iss"

Write-Host "Building Windows bundle in $bundleRoot"

if (Test-Path $bundleRoot) {
    Remove-Item $bundleRoot -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $bundleRoot | Out-Null

Push-Location $repoRoot
try {
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build -o $exePath .\cmd\cims

    Copy-Item .\templates -Destination $bundleRoot -Recurse
    Copy-Item .\static -Destination $bundleRoot -Recurse
    New-Item -ItemType Directory -Force -Path (Join-Path $bundleRoot "db") | Out-Null
    Copy-Item .\db\migrations -Destination (Join-Path $bundleRoot "db") -Recurse
    Copy-Item .\.env.example -Destination (Join-Path $bundleRoot ".env.example")
    Copy-Item .\README.md -Destination (Join-Path $bundleRoot "README.md")
    Copy-Item .\docs\windows.md -Destination (Join-Path $bundleRoot "WINDOWS.md")

    $launcher = @"
@echo off
setlocal
cd /d %~dp0
if not exist ".env" (
  copy ".env.example" ".env" >nul
)
echo Starting CIMS...
echo Edit .env if you need a different database or port.
cims.exe
"@
    Set-Content -Path (Join-Path $bundleRoot "start-cims.bat") -Value $launcher -Encoding ASCII

    $envFile = @"
ADDR=$Addr
DATABASE_URL=postgres://cims:cims@localhost:5432/cims?sslmode=disable
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123
SESSION_HASH_KEY=change-this-session-hash-key-32-bytes
SESSION_BLOCK_KEY=0123456789abcdef
"@
    Set-Content -Path (Join-Path $bundleRoot ".env") -Value $envFile -Encoding ASCII

    if (Test-Path $zipPath) {
        Remove-Item $zipPath -Force
    }
    Compress-Archive -Path (Join-Path $bundleRoot "*") -DestinationPath $zipPath
    Write-Host "ZIP package created: $zipPath"

    $iscc = $null
    $isccCmd = Get-Command iscc -ErrorAction SilentlyContinue
    if ($isccCmd) {
        $iscc = $isccCmd.Source
    } else {
        $commonPaths = @(
            "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
            "${env:ProgramFiles}\Inno Setup 6\ISCC.exe"
        )
        foreach ($candidate in $commonPaths) {
            if (Test-Path $candidate) {
                $iscc = $candidate
                break
            }
        }
    }

    if ($iscc) {
        & $iscc "/DAppVersion=$Version" "/DSourceBundle=$bundleRoot" $issPath
        Write-Host "Installer created in $distRoot"
    } else {
        Write-Host "Inno Setup not found. ZIP package created only."
    }
}
finally {
    Pop-Location
}
