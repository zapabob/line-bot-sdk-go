# Personal LINE Hakua Bridge startup installer
# User-level Windows logon autostart. No secrets are embedded.

param([switch]$Uninstall)

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent (Split-Path -Parent $scriptDir)
$startupFolder = [Environment]::GetFolderPath("Startup")
$vbsPath = Join-Path $startupFolder "HakuaPersonalLineBridge.vbs"
$logsDir = Join-Path $repoRoot "logs\personal_line_bridge"
$scriptPath = Join-Path $scriptDir "start_personal_line_bridge.py"

if ($Uninstall) {
    Write-Host "Removing Hakua Personal LINE Bridge startup..."
    if (Test-Path $vbsPath) { Remove-Item $vbsPath -Force }
    Write-Host "Done"
    exit 0
}

if (-not (Get-Command py -ErrorAction SilentlyContinue) -and -not (Get-Command python -ErrorAction SilentlyContinue)) {
    Write-Host "Python launcher not found"
    exit 1
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Go not found"
    exit 1
}
if (-not (Test-Path $logsDir)) { New-Item -ItemType Directory -Path $logsDir -Force | Out-Null }

$lines = @(
    'Set sh = CreateObject("WScript.Shell")'
    'Set fso = CreateObject("Scripting.FileSystemObject")'
    ('repoRoot = "{0}"' -f $repoRoot)
    'scriptPath = repoRoot & "\examples\personal_line_bridge\start_personal_line_bridge.py"'
    'logDir = repoRoot & "\logs\personal_line_bridge"'
    'If Not fso.FolderExists(logDir) Then fso.CreateFolder(logDir)'
    'logFile = logDir & "\startup_" & Year(Now) & Right("0" & Month(Now), 2) & Right("0" & Day(Now), 2) & ".log"'
    'sh.CurrentDirectory = repoRoot'
    'cmd = "cmd /c py -3 """ & scriptPath & """ >> """ & logFile & """ 2>&1"'
    'sh.Run cmd, 0, False'
)
$lines | Set-Content -Path $vbsPath -Encoding ASCII

Write-Host "Hakua Personal LINE Bridge startup configured"
Write-Host "Startup VBS: $vbsPath"
Write-Host "Logs:        $logsDir"
Write-Host "Start:       Windows logon background startup"
Write-Host "Uninstall: powershell -ExecutionPolicy Bypass -File `"$PSCommandPath`" -Uninstall"
