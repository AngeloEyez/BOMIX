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

# Check if cleanVer is purely numeric dot-separated (e.g. 1.0, 1.15.0, 2.0.1.0)
if ($cleanVer -match '^\d+(\.\d+)*$') {
  # Standardize file_version to at least Major.Minor.Patch.Build format or Major.Minor.Patch
  $parts = $cleanVer.Split('.')
  while ($parts.Count -lt 3) {
    $parts += "0"
  }
  $fileVer = $parts -join '.'
  $prodVer = $Version
} else {
  # Fallback for non-numeric versions like "dev", "main-latest", "feature-branch"
  $fileVer = "0.0.0.0"
  $prodVer = if ([string]::IsNullOrWhiteSpace($Version)) { "dev" } else { $Version }
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
