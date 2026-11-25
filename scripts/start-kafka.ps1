# Starts the local Kafka 3.9.2 broker in KRaft mode (localhost:9092), formatting storage on first run.
# Expects the Kafka folder next to the project folder: <parent>\kafka_2.13-3.9.2
$ErrorActionPreference = 'Stop'
$kafkaHome = Resolve-Path (Join-Path $PSScriptRoot '..\..\kafka_2.13-3.9.2')
$config    = Join-Path $kafkaHome 'config\kraft\server.properties'
$dataDir   = Join-Path $kafkaHome 'data\kraft-combined-logs'
Set-Location $kafkaHome
# Setting the heap here skips the 'wmic' OS check in kafka-server-start.bat (wmic is gone on newer Windows 11)
$env:KAFKA_HEAP_OPTS = '-Xms512M -Xmx1G'

if (-not (Test-Path (Join-Path $dataDir 'meta.properties'))) {
    Write-Host 'First run: formatting KRaft storage...'
    $clusterId = (& .\bin\windows\kafka-storage.bat random-uuid | Select-Object -Last 1).Trim()
    & .\bin\windows\kafka-storage.bat format -t $clusterId -c $config
}

Write-Host "Starting Kafka from $kafkaHome ... (Ctrl+C to stop)"
& .\bin\windows\kafka-server-start.bat $config
