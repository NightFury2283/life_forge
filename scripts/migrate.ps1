# Apply pending migrations without relying on Docker bind mounts of Windows paths.
$ErrorActionPreference = 'Stop'
$projectPath = Split-Path -Parent $PSScriptRoot
$settings = @{}
Get-Content -LiteralPath (Join-Path $projectPath '.env') | ForEach-Object {
    if ($_ -match '^\s*(POSTGRES_USER|POSTGRES_PASSWORD|POSTGRES_DB)\s*=\s*(.*)\s*$') {
        $settings[$Matches[1]] = $Matches[2].Trim().Trim('"').Trim("'")
    }
}
foreach ($key in @('POSTGRES_USER','POSTGRES_PASSWORD','POSTGRES_DB')) {
    $externalValue = [Environment]::GetEnvironmentVariable($key)
    if ($externalValue) { $settings[$key] = $externalValue }
    if (-not $settings[$key]) { throw "Missing $key" }
}
$migrationContainer = 'life-forge-migrate-' + [guid]::NewGuid().ToString('N').Substring(0,12)
$dbUser = [uri]::EscapeDataString($settings['POSTGRES_USER'])
$dbPassword = [uri]::EscapeDataString($settings['POSTGRES_PASSWORD'])
$dbName = [uri]::EscapeDataString($settings['POSTGRES_DB'])
$migrationDsn = "postgres://${dbUser}:${dbPassword}@life-forge-postgres:5432/${dbName}?sslmode=disable"
Push-Location $projectPath
try {
    docker compose run -d --no-deps --name $migrationContainer --entrypoint /bin/sleep life-forge-postgres-migrate 120
    if ($LASTEXITCODE -ne 0) { throw 'Cannot start migration container' }
    docker cp (Join-Path $projectPath 'migrations') "${migrationContainer}:/tmp/migrations"
    if ($LASTEXITCODE -ne 0) { throw 'Cannot copy migrations' }
    $result = & docker exec $migrationContainer migrate -path /tmp/migrations -database $migrationDsn up 2>&1
    $migrationExit = $LASTEXITCODE
    $result | ForEach-Object { $_.ToString().Replace($migrationDsn,'[database connection]').Replace($settings['POSTGRES_PASSWORD'],'[redacted]') }
    if ($migrationExit -ne 0) { throw 'Migration failed' }
} finally {
    docker rm -f $migrationContainer 2>$null | Out-Null
    Pop-Location
}
