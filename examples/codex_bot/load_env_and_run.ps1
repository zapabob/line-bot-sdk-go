# .envファイルを読み込んで環境変数を設定し、サーバーを起動
$env:Path += ";C:\Program Files\Go\bin"

# .envファイルが存在するか確認
if (Test-Path ".env") {
    Write-Host "📄 .envファイルを読み込み中..." -ForegroundColor Cyan
    
    # .envファイルの各行を読み込んで環境変数に設定
    Get-Content ".env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]*)=(.*)$') {
            $key = $matches[1].Trim()
            $value = $matches[2].Trim()
            # クォートを削除
            $value = $value -replace '^["'']|["'']$', ''
            [Environment]::SetEnvironmentVariable($key, $value, "Process")
            Write-Host "  ✓ $key を設定しました" -ForegroundColor Green
        }
    }
} else {
    Write-Host "⚠️ .envファイルが見つかりません" -ForegroundColor Yellow
}

# 環境変数の確認
Write-Host "`n🔍 環境変数の確認:" -ForegroundColor Cyan
Write-Host "  LINE_CHANNEL_SECRET: $($env:LINE_CHANNEL_SECRET -ne $null -and $env:LINE_CHANNEL_SECRET -ne '')" -ForegroundColor $(if ($env:LINE_CHANNEL_SECRET) { "Green" } else { "Red" })
Write-Host "  LINE_CHANNEL_ACCESS_TOKEN: $($env:LINE_CHANNEL_ACCESS_TOKEN -ne $null -and $env:LINE_CHANNEL_ACCESS_TOKEN -ne '')" -ForegroundColor $(if ($env:LINE_CHANNEL_ACCESS_TOKEN) { "Green" } else { "Red" })
Write-Host "  OPENAI_API_KEY: $($env:OPENAI_API_KEY -ne $null -and $env:OPENAI_API_KEY -ne '')" -ForegroundColor $(if ($env:OPENAI_API_KEY) { "Green" } else { "Yellow" })

# 必須環境変数のチェック
if (-not $env:LINE_CHANNEL_SECRET -or -not $env:LINE_CHANNEL_ACCESS_TOKEN) {
    Write-Host "`n❌ エラー: LINE_CHANNEL_SECRET と LINE_CHANNEL_ACCESS_TOKEN が設定されていません" -ForegroundColor Red
    Write-Host "   .envファイルに以下の設定を追加してください:" -ForegroundColor Yellow
    Write-Host "   LINE_CHANNEL_SECRET=your_channel_secret" -ForegroundColor Yellow
    Write-Host "   LINE_CHANNEL_ACCESS_TOKEN=your_channel_access_token" -ForegroundColor Yellow
    exit 1
}

Write-Host "`n🚀 サーバーを起動します..." -ForegroundColor Cyan
Write-Host "   Webhook URL: http://localhost:8080/webhook" -ForegroundColor Green
Write-Host "   Health Check: http://localhost:8080/health" -ForegroundColor Green
Write-Host "`n停止するには Ctrl+C を押してください`n" -ForegroundColor Yellow

# サーバーを起動
go run main.go webhook_template.go

