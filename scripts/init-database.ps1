# Creates TradeDB (schema, triggers) and loads seed data inside the Docker SQL Server.
# Safe to re-run: the scripts use IF NOT EXISTS guards.
$ErrorActionPreference = 'Continue'  # PS 5.1 treats native stderr as errors; exit codes are checked instead
$envFile = Join-Path $PSScriptRoot '..\backend\.env'
$saPassword = (Select-String -Path $envFile -Pattern '^MSSQL_PASSWORD=(.+)$').Matches[0].Groups[1].Value.Trim()
$dbDir = Resolve-Path (Join-Path $PSScriptRoot '..\database')
$name = 'tradeflow-sql'

foreach ($file in '01_schema.sql', '02_seed_data.sql') {
    Write-Host "Running $file ..."
    docker cp (Join-Path $dbDir $file) "${name}:/tmp/$file" | Out-Null
    docker exec $name /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P $saPassword -C -b -i "/tmp/$file"
    if ($LASTEXITCODE -ne 0) { throw "$file failed" }
}
docker exec $name /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P $saPassword -C -d TradeDB -Q "SELECT (SELECT COUNT(*) FROM client_details) AS clients, (SELECT COUNT(*) FROM stock_master) AS stocks, (SELECT COUNT(*) FROM trade_details) AS trades"
