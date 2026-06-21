# Codex Bot バックグラウンド自動起動設定スクリプト
# コマンドプロンプトウィンドウを表示しないバージョン

param(
    [switch]$Uninstall
)

# 現在のディレクトリを取得
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$startupFolder = [Environment]::GetFolderPath("Startup")
$vbsPath = Join-Path $scriptDir "start_codex_bot.vbs"
$shortcutPath = Join-Path $startupFolder "CodexBot.lnk"

if ($Uninstall) {
    # ショートカットとVBSファイルの削除
    Write-Host "🗑️  バックグラウンド自動起動設定を削除しています..." -ForegroundColor Cyan
    
    if (Test-Path $shortcutPath) {
        Remove-Item $shortcutPath -Force
    }
    if (Test-Path $vbsPath) {
        Remove-Item $vbsPath -Force
    }
    
    Write-Host "✅ 自動起動設定を削除しました" -ForegroundColor Green
    exit 0
}

# Goのパスを確認
$goPath = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $goPath) {
    Write-Host "❌ Go言語がインストールされていません" -ForegroundColor Red
    Write-Host "   まず Go をインストールしてください: https://go.dev/dl/"
    exit 1
}

# VBScriptファイルの作成（バックグラウンド実行用）
# VBScriptのパスをエスケープ
$escapedScriptDir = $scriptDir.Replace('\', '\\')

$vbsContent = @"
Set objShell = CreateObject("WScript.Shell")
Set objFSO = CreateObject("Scripting.FileSystemObject")

' ログファイルのパス
logDir = "$escapedScriptDir\logs"
If Not objFSO.FolderExists(logDir) Then
    objFSO.CreateFolder(logDir)
End If

logFile = logDir & "\codex_bot_" & Year(Now) & Right("0" & Month(Now), 2) & Right("0" & Day(Now), 2) & ".log"

' ログファイルを開く
Set objLogFile = objFSO.OpenTextFile(logFile, 8, True)
objLogFile.WriteLine "[" & Now & "] Starting Codex Bot Server..."

' コマンドを実行（非表示）
objShell.CurrentDirectory = "$escapedScriptDir"
objShell.Run "cmd /c cd /d `"$escapedScriptDir`" && go run main.go webhook_template.go >> `"" & logFile & "`" 2>&1", 0, False

objLogFile.WriteLine "[" & Now & "] Server started"
objLogFile.Close
"@

$vbsContent | Out-File -FilePath $vbsPath -Encoding ASCII -Force
Write-Host "✅ VBScriptファイルを作成しました: $vbsPath" -ForegroundColor Green

# ログディレクトリの作成
$logDir = Join-Path $scriptDir "logs"
if (-not (Test-Path $logDir)) {
    New-Item -ItemType Directory -Path $logDir | Out-Null
    Write-Host "✅ ログディレクトリを作成しました: $logDir" -ForegroundColor Green
}

# ショートカットの作成
Write-Host "📝 スタートアップショートカットを作成しています..." -ForegroundColor Cyan

$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = "wscript.exe"
$shortcut.Arguments = "`"$vbsPath`""
$shortcut.WorkingDirectory = $scriptDir
$shortcut.Description = "Codex Bot Webhook Server (Background)"
$shortcut.WindowStyle = 7  # 最小化
$shortcut.Save()

Write-Host "✅ バックグラウンド自動起動設定が完了しました！" -ForegroundColor Green
Write-Host ""
Write-Host "📋 設定内容:" -ForegroundColor Cyan
Write-Host "   ショートカット: $shortcutPath"
Write-Host "   VBScript: $vbsPath"
Write-Host "   ログディレクトリ: $logDir"
Write-Host "   起動タイミング: Windowsログオン時（バックグラウンド）"
Write-Host ""
Write-Host "💡 確認方法:" -ForegroundColor Yellow
Write-Host "   1. Win+R キーを押す"
Write-Host "   2. 'shell:startup' と入力してEnter"
Write-Host "   3. 'CodexBot.lnk' が表示されます"
Write-Host ""
Write-Host "📝 ログの確認:" -ForegroundColor Yellow
Write-Host "   ログファイル: $logDir\codex_bot_YYYYMMDD.log"
Write-Host ""
Write-Host "🔄 削除方法:" -ForegroundColor Yellow
Write-Host "   .\install_service_background.ps1 -Uninstall"
Write-Host ""
Write-Host "⚠️  注意: この方法では、ウィンドウは表示されずバックグラウンドで実行されます" -ForegroundColor Yellow
