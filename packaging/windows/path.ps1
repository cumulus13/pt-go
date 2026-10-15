# Adds/removes a directory to/from the SYSTEM (machine) PATH.
#  - Works on the registry value directly, so the existing REG_EXPAND_SZ type and
#    unexpanded %VARS% are preserved, and there is no 1024-char (NSIS/setx) truncation.
#  - Idempotent: 'add' never duplicates, 'remove' removes every matching entry.
#  - Never writes if nothing changed.
# Usage: powershell -NoProfile -ExecutionPolicy Bypass -File path.ps1 -Action add|remove -Dir C:\TOOLS\pt
param(
    [Parameter(Mandatory = $true)][ValidateSet('add', 'remove')][string]$Action,
    [Parameter(Mandatory = $true)][string]$Dir
)

$ErrorActionPreference = 'Stop'

function Normalize([string]$s) {
    return $s.Trim().Trim('"').TrimEnd('\').ToLowerInvariant()
}

$Dir = $Dir.Trim().TrimEnd('\')
if ([string]::IsNullOrWhiteSpace($Dir)) { throw 'Dir is empty' }
$target = Normalize $Dir

$base = [Microsoft.Win32.RegistryKey]::OpenBaseKey(
    [Microsoft.Win32.RegistryHive]::LocalMachine,
    [Microsoft.Win32.RegistryView]::Registry64)
$key = $base.OpenSubKey('SYSTEM\CurrentControlSet\Control\Session Manager\Environment', $true)
if ($null -eq $key) { throw 'Cannot open the system Environment registry key (run as Administrator)' }

try {
    $current = [string]$key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    $parts = @($current -split ';' | Where-Object { $_.Trim() -ne '' })

    $has = @($parts | Where-Object { (Normalize $_) -eq $target }).Count -gt 0

    if ($Action -eq 'add') {
        if ($has) { Write-Host "PATH already contains $Dir"; exit 0 }
        $parts = @($parts) + $Dir
        Write-Host "Added $Dir to system PATH"
    }
    else {
        if (-not $has) { Write-Host "PATH does not contain $Dir"; exit 0 }
        $parts = @($parts | Where-Object { (Normalize $_) -ne $target })
        Write-Host "Removed $Dir from system PATH"
    }

    $new = ($parts -join ';')
    $key.SetValue('Path', $new, [Microsoft.Win32.RegistryValueKind]::ExpandString)
}
finally {
    $key.Close()
    $base.Close()
}
exit 0
