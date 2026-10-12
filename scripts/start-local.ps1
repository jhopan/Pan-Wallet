$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Get-Content "$root\.env" | ForEach-Object {
  if ($_ -match '^\s*([A-Z_]+)\s*=\s*(.+)$') {
    [Environment]::SetEnvironmentVariable($matches[1], $matches[2].Trim())
  }
}

Write-Host "PAN_WALLET_ADDR=$env:PAN_WALLET_ADDR"
Start-Process -FilePath "$root\pan-wallet.exe" `
  -RedirectStandardOutput "$root\pan-wallet.out.log" `
  -RedirectStandardError "$root\pan-wallet.err.log" `
  -WindowStyle Hidden
Start-Sleep 3
Get-Content "$root\pan-wallet.err.log" -ErrorAction SilentlyContinue
