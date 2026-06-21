@echo off
chcp 65001 >nul
echo ========================================
echo Codex Bot Server 起動スクリプト
echo ========================================
echo.

REM Go言語の確認
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [エラー] Go言語がインストールされていません
    echo.
    echo インストール手順:
    echo   1. https://go.dev/dl/ にアクセス
    echo   2. Windows用のインストーラー（.msi）をダウンロード
    echo   3. インストーラーを実行
    echo   4. このバッチファイルを再実行
    echo.
    echo 詳細は setup_guide.txt を参照してください
    echo.
    pause
    exit /b 1
)

echo [OK] Go言語がインストールされています
go version
echo.

REM 環境変数の確認
if "%LINE_CHANNEL_SECRET%"=="" (
    echo [警告] LINE_CHANNEL_SECRET が設定されていません
)
if "%LINE_CHANNEL_ACCESS_TOKEN%"=="" (
    echo [警告] LINE_CHANNEL_ACCESS_TOKEN が設定されていません
)
if "%OPENAI_API_KEY%"=="" (
    echo [警告] OPENAI_API_KEY が設定されていません
)

echo.
echo ========================================
echo サーバーを起動しています...
echo ========================================
echo.

REM サーバーを起動
go run main.go webhook_template.go

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [エラー] サーバーの起動に失敗しました
    echo.
    pause
)
