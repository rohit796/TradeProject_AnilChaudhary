# Live view of SQL Server vs Elasticsearch, refreshed every 2 seconds. Ctrl+C to stop.
# Make a change in SQL in another window and watch it appear here within ~3 s.
#   powershell -ExecutionPolicy Bypass -File .\scripts\watch-sync.ps1
param([int]$Interval = 2, [switch]$Once)
$ErrorActionPreference = 'Continue'
$sql = Join-Path $PSScriptRoot 'sql.ps1'
$es  = 'http://localhost:9200/trade_reports'

function Get-SqlLatest {
    & $sql "SET NOCOUNT ON; SELECT TOP 5 trade_id, symbol, trade_type, quantity, price, CONVERT(varchar(19), updated_on, 120) AS updated_on FROM trade_details ORDER BY updated_on DESC" |
        Select-Object -Skip 2 | Where-Object { $_ -match '\S' -and $_ -notmatch 'rows affected' }
}

do {
    $sqlCount = ((& $sql "SET NOCOUNT ON; SELECT COUNT(*) FROM trade_details") | Select-Object -Skip 2 | Where-Object { $_ -match '^\s*\d+\s*$' } | Select-Object -First 1)
    if ($sqlCount) { $sqlCount = $sqlCount.Trim() }
    try { $esCount = (Invoke-RestMethod "$es/_count").count } catch { $esCount = 'DOWN' }
    try {
        $body = '{"size":5,"sort":[{"updated_on":"desc"}],"_source":["trade_id","symbol","trade_type","quantity","price","first_name","updated_on","indexed_at"]}'
        $esTop = (Invoke-RestMethod "$es/_search" -Method Post -ContentType 'application/json' -Body $body).hits.hits._source
    } catch { $esTop = @() }

    Clear-Host
    Write-Host ("TradeFlow live sync monitor    {0}    (Ctrl+C to stop)" -f (Get-Date -Format 'HH:mm:ss')) -ForegroundColor Cyan
    $color = if ("$sqlCount" -eq "$esCount") { 'Green' } else { 'Yellow' }
    Write-Host ("Trades in SQL Server: {0}    Trades in Elasticsearch: {1}    {2}" -f $sqlCount, $esCount, $(if ($color -eq 'Green') { 'IN SYNC' } else { 'DIFFERENT (deletes do not sync)' })) -ForegroundColor $color

    Write-Host "`nSQL Server - last 5 changed trades (updated_on):" -ForegroundColor White
    Get-SqlLatest | ForEach-Object { Write-Host "  $_" }

    Write-Host "`nElasticsearch - last 5 changed trades (what the dashboard shows):" -ForegroundColor White
    foreach ($t in $esTop) {
        $u = ([datetime]$t.updated_on).ToString('yyyy-MM-dd HH:mm:ss')
        $i = if ($t.indexed_at) { ([datetime]$t.indexed_at).ToString('HH:mm:ss') } else { '-' }
        Write-Host ("  {0,-14} {1,-9} {2,-4} qty {3,-5} price {4,-9} {5,-12} updated {6}  indexed {7}" -f $t.trade_id, $t.symbol, $t.trade_type, $t.quantity, $t.price, $t.first_name, $u, $i)
    }
    if (-not $Once) { Start-Sleep -Seconds $Interval }
} while (-not $Once)
