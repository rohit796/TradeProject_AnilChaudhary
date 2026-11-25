# Runs SQL Server 2022 (Developer edition) in Docker on localhost:1433 as an alternative
# to a local SQL Server install. Data lives in the Docker volume 'tradeflow-sql-data'.
# The sa password is read from backend\.env (MSSQL_PASSWORD).
# Running this script means you accept the SQL Server EULA: https://go.microsoft.com/fwlink/?linkid=857698
$ErrorActionPreference = 'Stop'
$envFile = Join-Path $PSScriptRoot '..\backend\.env'
$pwLine = Select-String -Path $envFile -Pattern '^MSSQL_PASSWORD=(.+)$' | Select-Object -First 1
if (-not $pwLine) { throw "MSSQL_PASSWORD is not set in $envFile" }
$saPassword = $pwLine.Matches[0].Groups[1].Value.Trim()

$name = 'tradeflow-sql'
$existing = docker ps -a --filter "name=^$name$" --format '{{.Status}}'
if ($existing) {
    if ($existing -notlike 'Up*') { docker start $name | Out-Null }
    Write-Host "Container '$name' is running."
} else {
    docker run -d --name $name `
        -e ACCEPT_EULA=Y -e MSSQL_PID=Developer -e "MSSQL_SA_PASSWORD=$saPassword" `
        -e TZ=Asia/Kolkata `
        -p 1433:1433 -v tradeflow-sql-data:/var/opt/mssql `
        --restart unless-stopped `
        mcr.microsoft.com/mssql/server:2022-latest | Out-Null
    Write-Host "Container '$name' created."
}

$ErrorActionPreference = 'Continue'  # sqlcmd errors are expected while SQL Server boots
Write-Host 'Waiting for SQL Server to accept connections...'
for ($i = 0; $i -lt 60; $i++) {
    docker exec $name /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P $saPassword -C -Q 'SELECT 1' -b *> $null
    if ($LASTEXITCODE -eq 0) { Write-Host 'SQL Server is ready on localhost,1433'; exit 0 }
    Start-Sleep -Seconds 2
}
throw 'SQL Server did not become ready in time. Check: docker logs tradeflow-sql'
