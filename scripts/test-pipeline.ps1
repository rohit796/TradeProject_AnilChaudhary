# End-to-end test of SQL Server -> (CDC poller) -> Kafka -> Elasticsearch -> API.
# Makes real changes in TradeDB, checks they reach Elasticsearch, then undoes them.
# All services and the backend must be running (scripts\start-all.ps1).
$ErrorActionPreference = 'Continue'
$sql = Join-Path $PSScriptRoot 'sql.ps1'
$api = 'http://localhost:8081/api'
$es  = 'http://localhost:9200/trade_reports'
$script:failed = 0

function Invoke-Sql($q) { & $sql $q | Out-Null }
function Get-EsDoc($id) { try { (Invoke-RestMethod "$es/_doc/$id")._source } catch { $null } }
function Assert-Within($label, [scriptblock]$check, $seconds = 12) {
    for ($i = 0; $i -lt $seconds; $i++) {
        if (& $check) { Write-Host "  PASS  $label" -ForegroundColor Green; return }
        Start-Sleep -Seconds 1
    }
    Write-Host "  FAIL  $label" -ForegroundColor Red; $script:failed++
}

Write-Host "`n0. Services"
$h = Invoke-RestMethod "$api/health"
Assert-Within 'backend up, SQL + Elasticsearch connected' { $h.database.connected -and $h.elasticsearch.connected } 1

Write-Host "`n1. INSERT a trade directly in SQL (CDC must stream it)"
Invoke-Sql "INSERT INTO trade_details (trade_id, order_id, client_id, symbol, trade_type, quantity, price, trade_value, net_amount) VALUES ('TRD-TEST-001', 'ORD-TEST-001', 'CL1001', 'TCS', 'BUY', 10, 4000, 40000, 40020)"
Assert-Within 'TRD-TEST-001 is in Elasticsearch with client + company filled in' { $d = Get-EsDoc 'TRD-TEST-001'; $d -and $d.client_code -eq 'ALOK001' -and $d.company_name -like 'Tata*' }

Write-Host "`n2. UPDATE the trade's price (trigger bumps updated_on, CDC re-sends it)"
Invoke-Sql "UPDATE trade_details SET price = 4200, trade_value = 42000, net_amount = 42020 WHERE trade_id = 'TRD-TEST-001'"
Assert-Within 'price is 4200 in Elasticsearch' { (Get-EsDoc 'TRD-TEST-001').price -eq 4200 }

Write-Host "`n3. UPDATE a client's name (all that client's trades must change in Elasticsearch)"
Invoke-Sql "UPDATE client_details SET first_name = 'AlokTest' WHERE client_id = 'CL1001'"
Assert-Within 'trades of CL1001 show first_name AlokTest' { (Get-EsDoc 'TRD-TEST-001').first_name -eq 'AlokTest' }
Invoke-Sql "UPDATE client_details SET first_name = 'Alok' WHERE client_id = 'CL1001'"
Assert-Within 'name reverted to Alok' { (Get-EsDoc 'TRD-TEST-001').first_name -eq 'Alok' }

Write-Host "`n4. UPDATE a stock's sector"
$stocks = Invoke-RestMethod "$api/stocks"   # assign first: PS 5.1 doesn't unroll a JSON array in a pipeline
$oldSector = ($stocks | Where-Object { $_.symbol -eq 'TCS' }).sector
if (-not $oldSector) { $oldSector = 'Information Technology' }
Invoke-Sql "UPDATE stock_master SET sector = 'IT Services Test' WHERE symbol = 'TCS'"
Assert-Within 'TCS trades show the new sector' { (Get-EsDoc 'TRD-TEST-001').sector -eq 'IT Services Test' }
Invoke-Sql "UPDATE stock_master SET sector = '$oldSector' WHERE symbol = 'TCS'"
Assert-Within 'sector reverted' { (Get-EsDoc 'TRD-TEST-001').sector -eq $oldSector }

Write-Host "`n5. Place a trade through the API (same call the New Trade button makes)"
$body = @{ trade_id = 'TRD-TEST-002'; client_id = 'CL1002'; symbol = 'RELIANCE'; exchange = 'NSE'; segment = 'EQUITY'; trade_type = 'SELL'; quantity = 5; price = 2900; brokerage = 20; trade_status = 'EXECUTED' } | ConvertTo-Json
$null = Invoke-RestMethod "$api/trades" -Method Post -ContentType 'application/json' -Body $body
Assert-Within 'TRD-TEST-002 is in SQL' { (& $sql "SELECT COUNT(*) AS n FROM trade_details WHERE trade_id = 'TRD-TEST-002'") -match '^\s*1\s*$' } 3
Assert-Within 'TRD-TEST-002 is in Elasticsearch' { Get-EsDoc 'TRD-TEST-002' }

Write-Host "`n6. Search and report API (reads from Elasticsearch)"
Assert-Within 'search=TEST finds both test trades' { (Invoke-RestMethod "$api/reports/trades?search=TEST").total -ge 2 } 3

Write-Host "`n7. Kafka topics"
Assert-Within 'trade-events, client-events, stock-events exist' { ((Invoke-RestMethod "$api/kafka/topics").topics.name -join ',') -match 'client-events.*stock-events.*trade-events' } 1

Write-Host "`nCleanup"
Invoke-Sql "DELETE FROM trade_details WHERE trade_id IN ('TRD-TEST-001', 'TRD-TEST-002')"
foreach ($id in 'TRD-TEST-001', 'TRD-TEST-002') { try { $null = Invoke-RestMethod "$es/_doc/$($id)?refresh=true" -Method Delete } catch {} }
Write-Host '  test trades removed from SQL and Elasticsearch'

if ($script:failed -eq 0) { Write-Host "`nALL TESTS PASSED" -ForegroundColor Green } else { Write-Host "`n$($script:failed) TEST(S) FAILED" -ForegroundColor Red; exit 1 }
