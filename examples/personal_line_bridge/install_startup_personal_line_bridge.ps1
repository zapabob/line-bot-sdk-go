# Personal LINE Hakua Bridge startup installer
# 管理者権限不要。Windowsログオン時に個人LINEブリッジ + Hakua返信Webhookをバックグラウンド起動します。

param([switch]$Uninstall)

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent (Split-Path -Parent $scriptDir)
$startupFolder = [Environment]::GetFolderPath("Startup")
$vbsPath = Join-Path $startupFolder "HakuaPersonalLineBridge.vbs"
$logsDir = Join-Path $repoRoot "logs\personal_line_bridge"
$scriptPath = Join-Path $scriptDir "start_personal_line_bridge.py"

if ($Uninstall) {
    Write-Host "🗑️  Hakua Personal LINE Bridge startup を削除します..." -ForegroundColor Cyan
    if (Test-Path $vbsPath) { Remove-Item $vbsPath -Force }
    Write-Host "✅ 削除完了" -ForegroundColor Green
    exit 0
}

if (-not (Get-Command py -ErrorAction SilentlyContinue) -and -not (Get-Command python -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Python launcher が見つかりません" -ForegroundColor Red
    exit 1
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Go が見つかりません" -ForegroundColor Red
    exit 1
}
if (-not (Test-Path $logsDir)) { New-Item -ItemType Directory -Path $logsDir -Force | Out-Null }

$lines = @(
    'Set sh = CreateObject("WScript.Shell")',
    'Set fso = CreateObject("Scripting.FileSystemObject")',
    'repoRoot = "' + $repoRoot + '"',
    'scriptPath = repoRoot & "\examples\personal_line_bridge\start_personal_line_bridge.py"',
    'logDir = repoRoot & "\logs\personal_line_bridge"',
    'If Not fso.FolderExists(logDir) Then fso.CreateFolder(logDir)',
    'logFile = logDir & "\startup_" & Year(Now) & Right("0" & Month(Now), 2) & Right("0" & Day(Now), 2) & ".log"',
    'sh.CurrentDirectory = repoRoot',
    'cmd = "cmd /c cd /d """ & repoRoot & """ && py -3 """ & scriptPath & """ >> """ & logFile & """ 2>&1"',
    'sh.Run cmd, 0, False'
)
$lines | Set-Content -Path $vbsPath -Encoding ASCII

Write-Host "✅ Hakua Personal LINE Bridge startup を設定しました" -ForegroundColor Green
Write-Host "   Startup VBS: $vbsPath"
Write-Host "   Logs:        $logsDir"
Write-Host "   起動:        Windowsログオン時（バックグラウンド）"
Write-Host "削除: powershell -ExecutionPolicy Bypass -File `"$PSCommandPath`" -Uninstall"
