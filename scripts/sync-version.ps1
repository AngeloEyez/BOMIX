# BOMIX Version Synchronization Script
param(
  [string]$Version = "dev",
  [string]$TargetPath = ""
)

$ErrorActionPreference = "Stop"

# Auto-detect info.json path
if ([string]::IsNullOrWhiteSpace($TargetPath) -or -not (Test-Path $TargetPath)) {
  $candidates = @(
    "$PSScriptRoot/../bomix-app/build/windows/info.json",
    "windows/info.json",
    "build/windows/info.json",
    "../build/windows/info.json"
  )
  foreach ($cand in $candidates) {
    if (Test-Path $cand) {
      $TargetPath = (Resolve-Path $cand).Path
      break
    }
  }
}

if (-not (Test-Path $TargetPath)) {
  Write-Warning "[BOMIX] info.json not found, skipping version sync"
  exit 0
}

# Normalize version string
$cleanVer = $Version.TrimStart("v").Trim()
if ($cleanVer -eq "dev" -or [string]::IsNullOrWhiteSpace($cleanVer)) {
  $fileVer = "0.0.0"
  $prodVer = "dev"
} else {
  $fileVer = $cleanVer
  $prodVer = $Version
}

try {
  $jsonContent = Get-Content -Path $TargetPath -Raw -Encoding UTF8 | ConvertFrom-Json
  $jsonContent.fixed.file_version = $fileVer
  $jsonContent.info.'0000'.ProductVersion = $prodVer
  $jsonString = $jsonContent | ConvertTo-Json -Depth 5
  $utf8NoBom = [System.Text.UTF8Encoding]::new($false)
  [System.IO.File]::WriteAllText($TargetPath, $jsonString, $utf8NoBom)
  Write-Host "[BOMIX] Windows metadata synchronized: file_version=$fileVer, ProductVersion=$prodVer"
} catch {
  Write-Error "[BOMIX] Failed to update $TargetPath : $_"
  exit 1
}
