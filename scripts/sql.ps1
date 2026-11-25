# Run SQL against TradeDB in the Docker SQL Server (password is read from backend\.env).
#
#   .\scripts\sql.ps1                                   -> interactive sqlcmd (type a query, then GO; type EXIT to quit)
#   .\scripts\sql.ps1 "SELECT * FROM stock_master"      -> run one query
#   .\scripts\sql.ps1 -File .\database\my_changes.sql   -> run a .sql file
param(
    [string]$File,
    # All remaining words form the query, so it works even if Command Prompt splits the quotes
    [Parameter(Position = 0, ValueFromRemainingArguments = $true)][string[]]$QueryParts
)
$Query = ($QueryParts -join ' ').Trim()
$ErrorActionPreference = 'Continue'
$envFile = Join-Path $PSScriptRoot '..\backend\.env'
$pw = (Select-String -Path $envFile -Pattern '^MSSQL_PASSWORD=(.+)$').Matches[0].Groups[1].Value.Trim()
$sqlcmd = '/opt/mssql-tools18/bin/sqlcmd'
$common = @('-S', 'localhost', '-U', 'sa', '-P', $pw, '-C', '-d', 'TradeDB')

if ($File) {
    $name = Split-Path $File -Leaf
    docker cp $File "tradeflow-sql:/tmp/$name" | Out-Null
    docker exec tradeflow-sql $sqlcmd @common -b -i "/tmp/$name"
} elseif ($Query) {
    docker exec tradeflow-sql $sqlcmd @common -b -W -s ' | ' -Q $Query
} else {
    Write-Host 'Connected to TradeDB. Type a query, press Enter, then type GO and Enter. EXIT to quit.'
    docker exec -it tradeflow-sql $sqlcmd @common
}
