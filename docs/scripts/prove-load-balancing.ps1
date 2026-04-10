param(
    [string]$Url = "http://localhost:8080/health",
    [int]$Requests = 10
)

Write-Host "Testing load balancing against: $Url"
Write-Host "Requests: $Requests"
Write-Host ""

$results = @()

for ($i = 1; $i -le $Requests; $i++) {
    $response = Invoke-RestMethod -Uri $Url -Method Get
    $instance = if ($null -ne $response.instance_name) { $response.instance_name } else { "unknown" }
    $results += $instance
    Write-Host ("Request {0} -> {1}" -f $i, $instance)
}

Write-Host ""
Write-Host "Summary:"
$results | Group-Object | Sort-Object Name | ForEach-Object {
    "{0,3} {1}" -f $_.Count, $_.Name
}
