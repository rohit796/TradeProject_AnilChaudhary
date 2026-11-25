# Creates the 3 TradeFlow topics (safe to re-run). Kafka must already be running.
$kafkaHome = Resolve-Path (Join-Path $PSScriptRoot '..\..\kafka_2.13-3.9.2')
Set-Location $kafkaHome
foreach ($t in 'trade-events', 'client-events', 'stock-events') {
    & .\bin\windows\kafka-topics.bat --create --topic $t --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1 --if-not-exists
}
& .\bin\windows\kafka-topics.bat --list --bootstrap-server localhost:9092
