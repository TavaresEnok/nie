$body = @{
    query = "comandos olt fiberhome"
    seed_urls = @("https://wiki.fiberhome.com")
    max_depth = 2
    max_urls = 50
    max_domains = 10
    max_bytes = 104857600
    min_domain_interval_seconds = 5
    use_web_discovery = $false
    use_git_discovery = $false
    use_mwmbl_discovery = $true
    use_swh_discovery = $false
} | ConvertTo-Json

$res = Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/campaigns" -Method Post -Body $body -ContentType "application/json"
$res | ConvertTo-Json -Depth 5
