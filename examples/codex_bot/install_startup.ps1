# Codex Bot スタートアップ設定スクリプト（簡単版）
# 管理者権限は不要です

param(
    [switch]$Uninstall
)

# 現在のディレクトリを取得
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$startupFolder = [Environment]::GetFolderPath("Startup")
$shortcutPath = Join-Path $startupFolder "CodexBot.lnk"

if ($Uninstall) {
    # ショートカットの削除
    Write-Host "🗑️  スタートアップ設定を削除しています..." -ForegroundColor Cyan
    
    if (Test-Path $shortcutPath) {
        Remove-Item $shortcutPath -Force
        Write-Host "✅ スタートアップ設定を削除しました" -ForegroundColor Green
    } else {
        Write-Host "ℹ️  スタートアップ設定が見つかりませんでした" -ForegroundColor Yellow
    }
    exit 0
}

# バッチファイルの作成
$batchFile = Join-Path $scriptDir "start_codex_bot.bat"
$batchContent = @"
@echo off
cd /d "$scriptDir"
echo Starting Codex Bot Server...
go run main.go webhook_template.go
pause
"@

$batchContent | Out-File -FilePath $batchFile -Encoding ASCII -Force
Write-Host "✅ 起動バッチファイルを作成しました: $batchFile" -ForegroundColor Green

# ショートカットの作成
Write-Host "📝 スタートアップショートカットを作成しています..." -ForegroundColor Cyan

$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = $batchFile
$shortcut.WorkingDirectory = $scriptDir
$shortcut.Description = "Codex Bot Webhook Server"
$shortcut.WindowStyle = 1  # 通常のウィンドウ
$shortcut.Save()

Write-Host "✅ スタートアップ設定が完了しました！" -ForegroundColor Green
Write-Host ""
Write-Host "📋 設定内容:" -ForegroundColor Cyan
Write-Host "   ショートカット: $shortcutPath"
Write-Host "   起動タイミング: Windowsログオン時"
Write-Host ""
Write-Host "💡 確認方法:" -ForegroundColor Yellow
Write-Host "   1. Win+R キーを押す"
Write-Host "   2. 'shell:startup' と入力してEnter"
Write-Host "   3. 'CodexBot.lnk' が表示されます"
Write-Host ""
Write-Host "🔄 削除方法:" -ForegroundColor Yellow
Write-Host "   .\install_startup.ps1 -Uninstall"
Write-Host ""
Write-Host "⚠️  注意: この方法では、ログイン時にコマンドプロンプトウィンドウが表示されます" -ForegroundColor Yellow
