# Starts the local Elasticsearch 9.5.7 node (http://localhost:9200).
# Expects the Elasticsearch folder next to the project folder:
#   <parent>\elasticsearch-9.5.7
#   <parent>\TradeProject_AnilChaudhary-main
$ErrorActionPreference = 'Stop'
$esHome = Resolve-Path (Join-Path $PSScriptRoot '..\..\elasticsearch-9.5.7')
# Use the JDK that ships inside Elasticsearch, never the system Java 8
$env:ES_JAVA_HOME = Join-Path $esHome 'jdk'
Write-Host "Starting Elasticsearch from $esHome ... (Ctrl+C to stop)"
& (Join-Path $esHome 'bin\elasticsearch.bat')
