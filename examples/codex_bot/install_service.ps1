# Codex Bot 自動起動設定スクリプト
# 管理者権限で実行してください

param(
    [switch]$Uninstall
)

# 管理者権限チェック
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "❌ このスクリプトは管理者権限で実行する必要があります" -ForegroundColor Red
    Write-Host ""
    Write-Host "PowerShellを管理者として実行してから、再度実行してください:" -ForegroundColor Yellow
    Write-Host "   1. PowerShellを右クリック"
    Write-Host "   2. '管理者として実行'を選択"
    Write-Host "   3. このスクリプトを実行"
    exit 1
}

# 現在のディレクトリを取得
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
$projectRoot = Split-Path -Parent $projectRoot

# 実行ファイルのパス
$exePath = Join-Path $scriptDir "codex-bot.exe"
$goPath = (Get-Command go -ErrorAction SilentlyContinue).Source

if (-not $goPath) {
    Write-Host "❌ Go言語がインストールされていません" -ForegroundColor Red
    Write-Host "   まず Go をインストールしてください: https://go.dev/dl/"
    exit 1
}

# 環境変数の確認
$requiredVars = @("LINE_CHANNEL_SECRET", "LINE_CHANNEL_ACCESS_TOKEN", "OPENAI_API_KEY")
$missingVars = @()

foreach ($var in $requiredVars) {
    $value = [Environment]::GetEnvironmentVariable($var, "Machine")
    if ([string]::IsNullOrEmpty($value)) {
        $missingVars += $var
    }
}

if ($missingVars.Count -gt 0) {
    Write-Host "⚠️  以下の環境変数がシステム環境変数に設定されていません:" -ForegroundColor Yellow
    foreach ($var in $missingVars) {
        Write-Host "   - $var"
    }
    Write-Host ""
    Write-Host "システム環境変数に設定しますか？ (Y/N)" -ForegroundColor Yellow
    $response = Read-Host
    
    if ($response -eq "Y" -or $response -eq "y") {
        foreach ($var in $missingVars) {
            Write-Host "   $var の値を入力してください:" -ForegroundColor Cyan
            $value = Read-Host
            [System.Environment]::SetEnvironmentVariable($var, $value, "Machine")
            Write-Host "   ✅ $var を設定しました" -ForegroundColor Green
        }
        Write-Host ""
        Write-Host "⚠️  環境変数の変更を反映するため、再起動が必要な場合があります" -ForegroundColor Yellow
    }
}

# タスク名
$taskName = "CodexBotService"

if ($Uninstall) {
    # タスクの削除
    Write-Host "🗑️  自動起動設定を削除しています..." -ForegroundColor Cyan
    
    $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($task) {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
        Write-Host "✅ 自動起動設定を削除しました" -ForegroundColor Green
    } else {
        Write-Host "ℹ️  自動起動設定が見つかりませんでした" -ForegroundColor Yellow
    }
    exit 0
}

# ビルド（実行ファイルが存在しない場合）
if (-not (Test-Path $exePath)) {
    Write-Host "🔨 実行ファイルをビルドしています..." -ForegroundColor Cyan
    Push-Location $scriptDir
    go build -o codex-bot.exe main.go webhook_template.go
    Pop-Location
    
    if (-not (Test-Path $exePath)) {
        Write-Host "❌ ビルドに失敗しました" -ForegroundColor Red
        exit 1
    }
    Write-Host "✅ ビルドが完了しました" -ForegroundColor Green
}

# タスクの作成
Write-Host "📝 自動起動タスクを作成しています..." -ForegroundColor Cyan

# 既存のタスクを削除（存在する場合）
$existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    Write-Host "   ℹ️  既存のタスクを削除しました" -ForegroundColor Yellow
}

# タスクアクション
$action = New-ScheduledTaskAction -Execute $exePath -WorkingDirectory $scriptDir

# タスクトリガー（ログオン時）
$trigger = New-ScheduledTaskTrigger -AtLogOn

# タスク設定
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -RestartCount 3 `
    -RestartInterval (New-TimeSpan -Minutes 1)

# タスクの登録
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -LogonType Interactive -RunLevel Highest

try {
    Register-ScheduledTask `
        -TaskName $taskName `
        -Action $action `
        -Trigger $trigger `
        -Settings $settings `
        -Principal $principal `
        -Description "Codex Bot Webhook Server - Auto start on Windows login" | Out-Null
    
    Write-Host "✅ 自動起動設定が完了しました！" -ForegroundColor Green
    Write-Host ""
    Write-Host "📋 設定内容:" -ForegroundColor Cyan
    Write-Host "   タスク名: $taskName"
    Write-Host "   実行ファイル: $exePath"
    Write-Host "   起動タイミング: Windowsログオン時"
    Write-Host ""
    Write-Host "💡 確認方法:" -ForegroundColor Yellow
    Write-Host "   1. タスクスケジューラーを開く"
    Write-Host "   2. 'タスク スケジューラー ライブラリ'を確認"
    Write-Host "   3. '$taskName' タスクが表示されます"
    Write-Host ""
    Write-Host "🔄 削除方法:" -ForegroundColor Yellow
    Write-Host "   .\install_service.ps1 -Uninstall"
    
} catch {
    Write-Host "❌ タスクの登録に失敗しました: $_" -ForegroundColor Red
    exit 1
}
