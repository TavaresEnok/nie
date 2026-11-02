$env:NIE_DATABASE_URL = "postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable"
$env:NIE_ADDR = "127.0.0.1:8080"
$env:NIE_DATA_DIR = "./data"
$env:NIE_WORKER = "true"
$env:NIE_WORKERS = "2"
$env:NIE_MWMBL_URL = "https://mwmbl.org"
$env:NIE_WAYBACK_ENABLED = "true"
$env:NIE_SOFTWARE_HERITAGE_ENABLED = "true"
$env:NIE_MENTIONS_DISCOVERY_ENABLED = "true"
$env:NIE_LOG_LEVEL = "info"

Write-Host "Iniciando NIE em http://127.0.0.1:8080 com banco em $ip:5432 ..."
& .\bin\nie.exe
