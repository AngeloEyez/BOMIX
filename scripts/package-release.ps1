<#
.SYNOPSIS
    BOMIX 發布包打包腳本 (Release Packaging Script)
.DESCRIPTION
    將 Windows 執行檔 (BOMIX.exe) 與最新出版級手冊 (BOMIX-使用者手冊.pdf)
    共同封裝為 ZIP 發布壓縮包。
    【重要原則】嚴格僅打包 .exe 與 .pdf 檔案，絕對不打包 .html 檔案。
#>
param(
  [string]$Version = "",
  [string]$OutputDir = ""
)

$ErrorActionPreference = "Stop"

# 取得專案根目錄
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = (Resolve-Path "$scriptDir/..").Path
if ([string]::IsNullOrWhiteSpace($OutputDir)) {
  $OutputDir = $rootDir
}

# 判定版本與壓縮檔名稱
if ([string]::IsNullOrWhiteSpace($Version)) {
  if ($env:VERSION) {
    $Version = $env:VERSION
  } else {
    $Version = "dev"
  }
}

$zipTag = if ($Version.StartsWith("v")) { $Version } else { "v$Version" }
$zipName = "BOMIX_${zipTag}.zip"
$zipPath = Join-Path $OutputDir $zipName

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "  BOMIX 發布壓縮包打包作業 (Release Packaging)" -ForegroundColor Cyan
Write-Host "  版本號: $Version" -ForegroundColor Cyan
Write-Host "  輸出目標: $zipPath" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

# 1. 檢查主程式執行檔
$exePath = Join-Path $rootDir "bomix-app/bin/BOMIX.exe"
if (-not (Test-Path $exePath)) {
  Write-Error "錯誤：未找到編譯輸出檔案 $exePath。請先執行 task build 或 wails3 build！"
  exit 1
}

# 2. 檢查或建置手冊 PDF
$pdfPath = Join-Path $rootDir "manual/dist/BOMIX-使用者手冊.pdf"
if (-not (Test-Path $pdfPath)) {
  Write-Host "未找到手冊 PDF，正在嘗試自動建置..." -ForegroundColor Yellow
  $manualDir = Join-Path $rootDir "manual"
  Push-Location $manualDir
  try {
    node build.js
  } finally {
    Pop-Location
  }
}

if (-not (Test-Path $pdfPath)) {
  Write-Error "錯誤：未找到手冊 PDF 檔案 $pdfPath！"
  exit 1
}

# 3. 建立乾淨的暫存打包資料夾
$stagingDir = Join-Path $rootDir "release-staging"
if (Test-Path $stagingDir) {
  Remove-Item -Path $stagingDir -Recurse -Force
}
New-Item -ItemType Directory -Path $stagingDir | Out-Null

# 複製目標檔案 (僅 .exe 與 .pdf)
Copy-Item -Path $exePath -Destination $stagingDir
Copy-Item -Path $pdfPath -Destination $stagingDir

# 4. 安全防呆檢查：確保絕無 HTML 檔案
$stagedFiles = Get-ChildItem -Path $stagingDir
Write-Host "`n打包內容清單：" -ForegroundColor Green
foreach ($file in $stagedFiles) {
  $fileMB = [math]::Round($file.Length / 1MB, 2)
  Write-Host "  - $($file.Name) ($fileMB MB)"
}

$htmlFiles = @($stagedFiles | Where-Object { $_.Extension -match '^\.html?$' })
if ($htmlFiles.Count -gt 0) {
  Write-Error "安全防呆失敗：打包目錄中含有 HTML 檔案，違反發布打包規範！"
  exit 1
}

# 5. 執行壓縮封裝
if (Test-Path $zipPath) {
  Remove-Item -Path $zipPath -Force
}
Compress-Archive -Path "$stagingDir/*" -DestinationPath $zipPath -Force

# 清理暫存資料夾
Remove-Item -Path $stagingDir -Recurse -Force

$zipItem = Get-Item $zipPath
$zipMB = [math]::Round($zipItem.Length / 1MB, 2)
Write-Host "`n🎉 打包完成！" -ForegroundColor Green
Write-Host "產出檔案: $($zipItem.FullName) ($zipMB MB)" -ForegroundColor Green
