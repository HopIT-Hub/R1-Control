# Build a Windows .zip package with the exe and libusb DLL.
#
# Usage: .\packaging\windows\build.ps1 -Binary r1ptt-windows-amd64.exe -Version 1.0.0
#
# Requires: libusb-1.0.dll available (from vcpkg or manual install)

param(
    [Parameter(Mandatory=$true)]
    [string]$Binary,

    [Parameter(Mandatory=$true)]
    [string]$Version
)

$Version = $Version -replace '^v', ''  # strip leading v so filename is R1-Control-v1.0.0 not vv1.0.0

$ErrorActionPreference = "Stop"

$AppName = "R1 Control"
$ZipName = "R1-Control-v${Version}-windows-amd64.zip"
$StageDir = "R1-Control-windows"

Write-Host "==> Building ${AppName} v${Version} (windows-amd64)"

# Clean
if (Test-Path $StageDir) { Remove-Item -Recurse -Force $StageDir }
if (Test-Path $ZipName) { Remove-Item -Force $ZipName }

# Create staging directory
New-Item -ItemType Directory -Path $StageDir | Out-Null

# Copy and rename binary
Copy-Item $Binary "${StageDir}\R1 Control.exe"

# Find and copy libusb DLL
$LibusbPaths = @(
    "C:\vcpkg\installed\x64-windows\bin\libusb-1.0.dll",
    "$env:VCPKG_INSTALLATION_ROOT\installed\x64-windows\bin\libusb-1.0.dll",
    ".\libusb-1.0.dll"
)

$LibusbFound = $false
foreach ($path in $LibusbPaths) {
    if (Test-Path $path) {
        Write-Host "==> Bundling libusb from ${path}"
        Copy-Item $path "${StageDir}\libusb-1.0.dll"
        $LibusbFound = $true
        break
    }
}

if (-not $LibusbFound) {
    Write-Warning "libusb-1.0.dll not found, zip will not include it"
}

# Create zip
Write-Host "==> Creating ${ZipName}"
Compress-Archive -Path "${StageDir}\*" -DestinationPath $ZipName -Force

# Cleanup staging
Remove-Item -Recurse -Force $StageDir

Write-Host "==> Done: ${ZipName}"
