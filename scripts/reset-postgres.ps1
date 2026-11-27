# Reset PostgreSQL Password & Setup Databases for BuyWise
$ErrorActionPreference = "Stop"

Write-Host "Configuring PostgreSQL for BuyWise..." -ForegroundColor Cyan

$hba = "C:\Program Files\PostgreSQL\18\data\pg_hba.conf"
if (-not (Test-Path $hba)) {
    Write-Error "Could not find pg_hba.conf at $hba"
    exit 1
}

# 1. Enable trust authentication temporarily
(Get-Content $hba) -replace 'host\s+all\s+all\s+(127\.0\.0\.1/32|::1/128)\s+scram-sha-256', 'host    all             all             $1            trust' | Set-Content $hba
Write-Host "Restarting PostgreSQL with temporary trust..." -ForegroundColor Yellow
Restart-Service postgresql-x64-18

# 2. Reset password to 'postgres'
Write-Host "Setting password for user 'postgres' to 'postgres'..." -ForegroundColor Yellow
psql -U postgres -h 127.0.0.1 -c "ALTER USER postgres PASSWORD 'postgres';"

# 3. Create databases if not exists
Write-Host "Creating databases 'selfemi' and 'selfemi_test'..." -ForegroundColor Yellow
psql -U postgres -h 127.0.0.1 -c "SELECT 'CREATE DATABASE selfemi' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'selfemi')\gexec"
psql -U postgres -h 127.0.0.1 -c "SELECT 'CREATE DATABASE selfemi_test' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'selfemi_test')\gexec"

# 4. Revert to scram-sha-256 authentication
(Get-Content $hba) -replace 'host\s+all\s+all\s+(127\.0\.0\.1/32|::1/128)\s+trust', 'host    all             all             $1            scram-sha-256' | Set-Content $hba
Write-Host "Restoring PostgreSQL security settings and restarting..." -ForegroundColor Yellow
Restart-Service postgresql-x64-18

Write-Host "`nPostgreSQL setup complete!" -ForegroundColor Green
Write-Host "Password set to: postgres" -ForegroundColor Green
Write-Host "Databases created: selfemi, selfemi_test" -ForegroundColor Green
Write-Host "Press Enter to exit..."
Read-Host
