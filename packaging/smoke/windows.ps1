# Smoke test on a Windows runner: silent install -> verify dir + SYSTEM PATH -> uninstall -> verify clean.
# Usage: powershell -File packaging/smoke/windows.ps1 -Dist dist -ExpectVersion 1.0.81
param(
    [Parameter(Mandatory = $true)][string]$Dist,
    [Parameter(Mandatory = $true)][string]$ExpectVersion
)
$ErrorActionPreference = 'Stop'
$installDir = 'C:\TOOLS\pt'

function Machine-Path { [Environment]::GetEnvironmentVariable('Path', 'Machine') }
function Path-Has([string]$dir) {
    $d = $dir.TrimEnd('\').ToLowerInvariant()
    return @((Machine-Path) -split ';' | Where-Object { $_.TrimEnd('\').ToLowerInvariant() -eq $d }).Count
}
function Check([bool]$cond, [string]$msg) {
    if (-not $cond) { throw "SMOKE FAIL: $msg" }
    Write-Host "ok: $msg"
}

$setup = Get-ChildItem -Path $Dist -Filter 'pt-*-windows-amd64-setup.exe' | Select-Object -First 1
if ($null -eq $setup) { throw "no installer found in $Dist" }
Write-Host "Installer: $($setup.FullName)"

$before = Machine-Path

# --- install (twice, to prove PATH is not duplicated on upgrade/reinstall)
foreach ($i in 1, 2) {
    $p = Start-Process -FilePath $setup.FullName -ArgumentList '/S' -Wait -PassThru
    Check ($p.ExitCode -eq 0) "silent install #$i exit code 0 (got $($p.ExitCode))"
}
Check (Test-Path "$installDir\pt.exe") 'pt.exe installed to C:\TOOLS\pt'
Check (Test-Path "$installDir\uninstall.exe") 'uninstaller present'
Check ((Path-Has $installDir) -eq 1) 'C:\TOOLS\pt present exactly once in SYSTEM PATH'

$out = (& "$installDir\pt.exe" --version) | Select-Object -First 1
Write-Host $out
Check ($out -eq "PT version $ExpectVersion") "pt.exe --version == $ExpectVersion"

$arp = Get-ItemProperty 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\pt-go' -ErrorAction SilentlyContinue
Check ($null -ne $arp -and $arp.DisplayVersion -eq $ExpectVersion) 'Add/Remove Programs entry present'

# --- uninstall (_?= makes the uninstaller run synchronously in place)
$u = Start-Process -FilePath "$installDir\uninstall.exe" -ArgumentList '/S', "_?=$installDir" -Wait -PassThru
Check ($u.ExitCode -eq 0) "silent uninstall exit code 0 (got $($u.ExitCode))"
Check (-not (Test-Path "$installDir\pt.exe")) 'pt.exe removed'
Check ((Path-Has $installDir) -eq 0) 'C:\TOOLS\pt removed from SYSTEM PATH'
Check ($null -eq (Get-ItemProperty 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\pt-go' -ErrorAction SilentlyContinue)) 'Add/Remove Programs entry removed'

# uninstall.exe stays when run with _?=, clean it up
Remove-Item "$installDir\uninstall.exe" -Force -ErrorAction SilentlyContinue
Remove-Item $installDir -Force -ErrorAction SilentlyContinue

# PATH must hold exactly the same entries as before (path.ps1 drops empty segments, so compare entry lists)
function Entries([string]$p) { @($p -split ';' | Where-Object { $_.Trim() -ne '' } | ForEach-Object { $_.Trim() }) }
Check (((Entries (Machine-Path)) -join ';') -eq ((Entries $before) -join ';')) 'SYSTEM PATH entries restored to the original list'
Write-Host 'WINDOWS SMOKE TEST OK'
