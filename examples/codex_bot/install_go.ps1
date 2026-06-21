# Go言語のインストールスクリプト

Write-Host "🔍 Go言語のインストール状況を確認しています..." -ForegroundColor Cyan

# Goのバージョンを確認
try {
    $goVersion = go version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Go言語は既にインストールされています: $goVersion" -ForegroundColor Green
        Write-Host ""
        Write-Host "次のステップ:" -ForegroundColor Yellow
        Write-Host "   1. 環境変数を設定してください"
        Write-Host "   2. go run main.go webhook_template.go でサーバーを起動"
        exit 0
    }
} catch {
    # Goが見つからない場合は続行
}

Write-Host "❌ Go言語がインストールされていません" -ForegroundColor Red
Write-Host ""

# Chocolateyが利用可能か確認
$chocoAvailable = $false
try {
    $chocoVersion = choco --version 2>&1
    if ($LASTEXITCODE -eq 0) {
        $chocoAvailable = $true
        Write-Host "✅ Chocolateyが利用可能です" -ForegroundColor Green
    }
} catch {
    # Chocolateyが利用できない場合は続行
}

Write-Host ""
Write-Host "📥 Go言語のインストール方法:" -ForegroundColor Yellow
Write-Host ""

if ($chocoAvailable) {
    Write-Host "方法1: Chocolateyを使用（推奨）" -ForegroundColor Cyan
    Write-Host "   以下のコマンドを管理者として実行してください:" -ForegroundColor White
    Write-Host "   choco install golang -y" -ForegroundColor Green
    Write-Host ""
    Write-Host "Chocolateyでインストールしますか？ (Y/N)" -ForegroundColor Yellow
    $response = Read-Host
    
    if ($response -eq "Y" -or $response -eq "y") {
        $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        
        if (-not $isAdmin) {
            Write-Host "❌ この操作には管理者権限が必要です" -ForegroundColor Red
            Write-Host "   PowerShellを管理者として実行してから、再度実行してください" -ForegroundColor Yellow
            exit 1
        }
        
        Write-Host "📦 ChocolateyでGoをインストールしています..." -ForegroundColor Cyan
        choco install golang -y
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Go言語のインストールが完了しました！" -ForegroundColor Green
            Write-Host ""
            Write-Host "⚠️  PowerShellを再起動してから、go version で確認してください" -ForegroundColor Yellow
        } else {
            Write-Host "❌ インストールに失敗しました" -ForegroundColor Red
        }
    }
} else {
    Write-Host "方法1: 公式インストーラーを使用（推奨）" -ForegroundColor Cyan
    Write-Host "   1. https://go.dev/dl/ にアクセス" -ForegroundColor White
    Write-Host "   2. Windows用のインストーラー（.msi）をダウンロード" -ForegroundColor White
    Write-Host "   3. インストーラーを実行" -ForegroundColor White
    Write-Host "   4. PowerShellを再起動" -ForegroundColor White
    Write-Host ""
    Write-Host "ブラウザでダウンロードページを開きますか？ (Y/N)" -ForegroundColor Yellow
    $response = Read-Host
    
    if ($response -eq "Y" -or $response -eq "y") {
        Start-Process "https://go.dev/dl/"
        Write-Host "✅ ブラウザでダウンロードページを開きました" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "方法2: Scoopを使用" -ForegroundColor Cyan
Write-Host "   scoop install go" -ForegroundColor Green
Write-Host ""
Write-Host "方法3: 手動インストール" -ForegroundColor Cyan
Write-Host "   1. https://go.dev/dl/ からダウンロード" -ForegroundColor White
Write-Host "   2. インストーラーを実行" -ForegroundColor White
Write-Host "   3. 環境変数PATHに C:\Program Files\Go\bin が自動的に追加されます" -ForegroundColor White
Write-Host ""
Write-Host "📚 詳細は WINDOWS_SETUP.md を参照してください" -ForegroundColor Cyan
