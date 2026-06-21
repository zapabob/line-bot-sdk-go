# Go言語のインストール確認スクリプト

Write-Host "🔍 Go言語のインストール状況を確認しています..." -ForegroundColor Cyan

# Goのバージョンを確認
try {
    $goVersion = go version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Go言語がインストールされています: $goVersion" -ForegroundColor Green
        Write-Host ""
        Write-Host "📝 次のステップ:" -ForegroundColor Yellow
        Write-Host "   1. 環境変数を設定してください"
        Write-Host "   2. go run main.go webhook_template.go でサーバーを起動"
    } else {
        throw "Go not found"
    }
} catch {
    Write-Host "❌ Go言語がインストールされていません" -ForegroundColor Red
    Write-Host ""
    Write-Host "📥 インストール方法:" -ForegroundColor Yellow
    Write-Host "   1. https://go.dev/dl/ にアクセス"
    Write-Host "   2. Windows用のインストーラー（.msi）をダウンロード"
    Write-Host "   3. インストーラーを実行"
    Write-Host "   4. PowerShellを再起動"
    Write-Host ""
    Write-Host "または、Chocolateyを使用:" -ForegroundColor Yellow
    Write-Host "   choco install golang"
    Write-Host ""
    Write-Host "詳細は WINDOWS_SETUP.md を参照してください"
}

Write-Host ""
Write-Host "🔍 環境変数の確認:" -ForegroundColor Cyan

$requiredVars = @(
    "LINE_CHANNEL_SECRET",
    "LINE_CHANNEL_ACCESS_TOKEN",
    "OPENAI_API_KEY"
)

$missingVars = @()
foreach ($var in $requiredVars) {
    $value = [Environment]::GetEnvironmentVariable($var, "Process")
    if ([string]::IsNullOrEmpty($value)) {
        $value = [Environment]::GetEnvironmentVariable($var, "User")
    }
    if ([string]::IsNullOrEmpty($value)) {
        $value = [Environment]::GetEnvironmentVariable($var, "Machine")
    }
    
    if ([string]::IsNullOrEmpty($value)) {
        Write-Host "  ❌ $var : 設定されていません" -ForegroundColor Red
        $missingVars += $var
    } else {
        $masked = if ($var -like "*KEY*" -or $var -like "*SECRET*" -or $var -like "*TOKEN*") {
            $value.Substring(0, [Math]::Min(10, $value.Length)) + "***"
        } else {
            $value
        }
        Write-Host "  ✅ $var : $masked" -ForegroundColor Green
    }
}

if ($missingVars.Count -gt 0) {
    Write-Host ""
    Write-Host "⚠️  以下の環境変数を設定してください:" -ForegroundColor Yellow
    foreach ($var in $missingVars) {
        Write-Host "   $var"
    }
    Write-Host ""
    Write-Host "設定方法:" -ForegroundColor Yellow
    Write-Host '   $env:' + $missingVars[0] + ' = "your_value"'
    Write-Host ""
    Write-Host "または、.envファイルを使用（godotenvが必要）"
}

Write-Host ""
Write-Host "📚 詳細なセットアップ手順は WINDOWS_SETUP.md を参照してください" -ForegroundColor Cyan
