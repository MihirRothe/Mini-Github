# ForgeHub Local Development Launcher
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "   Starting ForgeHub Development Stack    " -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

Write-Host "[1/3] Checking environment..." -ForegroundColor Yellow
$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    if (Test-Path "C:\Program Files\Go\bin\go.exe") {
        $env:Path += ";C:\Program Files\Go\bin"
    } else {
        Write-Error "Go is not found in PATH or standard installation directory."
        exit 1
    }
}

Write-Host "[2/3] Starting ForgeHub API on http://localhost:8080..." -ForegroundColor Green
$apiJob = Start-Job -ScriptBlock {
    param($dir)
    Set-Location "$dir\apps\api"
    & go run cmd/server/main.go
} -ArgumentList $projectRoot

Write-Host "[3/3] Starting ForgeHub Web on http://localhost:5173..." -ForegroundColor Green
Set-Location "$projectRoot\apps\web"
if (-not (Test-Path "node_modules")) {
    Write-Host "Installing web dependencies..." -ForegroundColor Gray
    npm install
}

npm run dev
