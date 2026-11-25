# Starts the whole TradeFlow stack, each service in its own PowerShell window:
#   Elasticsearch (9200) -> Kafka (9092) -> SQL Server in Docker (1433) -> Go backend (8081) -> React UI (5173)
# Requires Docker Desktop to be running.
$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..')

function Start-Window($title, $command) {
    Start-Process powershell -ArgumentList '-NoExit', '-ExecutionPolicy', 'Bypass', '-Command', "`$host.UI.RawUI.WindowTitle = '$title'; $command"
}

function Wait-Port($port, $name, $seconds = 120) {
    Write-Host "Waiting for $name on port $port ..."
    for ($i = 0; $i -lt $seconds; $i++) {
        if (Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue) { Write-Host "  $name is up"; return }
        Start-Sleep -Seconds 1
    }
    throw "$name did not start on port $port"
}

Start-Window 'Elasticsearch' "& '$root\scripts\start-elasticsearch.ps1'"
Start-Window 'Kafka' "& '$root\scripts\start-kafka.ps1'"
& "$root\scripts\start-sqlserver.ps1"
Wait-Port 9200 'Elasticsearch'
Wait-Port 9092 'Kafka'

Start-Window 'TradeFlow Backend' "Set-Location '$root\backend'; if (-not (Test-Path .\trade-backend.exe)) { go build -o trade-backend.exe main.go }; .\trade-backend.exe"
Wait-Port 8081 'Backend API'
Start-Window 'TradeFlow Frontend' "Set-Location '$root\frontend'; npm run dev"

Write-Host ''
Write-Host 'All services started. Open http://localhost:5173 (login from frontend\.env).'
Write-Host 'First time only: load reports with  Invoke-RestMethod http://localhost:8081/api/sync/trades -Method Post'
