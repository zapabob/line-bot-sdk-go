# Windows環境でのセットアップガイド

Windows環境でCodex Botを実行するための手順です。

## 🔧 Go言語のインストール

### 方法1: 公式インストーラー（推奨）

1. [Go公式サイト](https://go.dev/dl/) にアクセス
2. Windows用のインストーラー（`.msi`ファイル）をダウンロード
3. インストーラーを実行してインストール
4. インストール後、PowerShellを再起動

### 方法2: Chocolateyを使用

```powershell
# Chocolateyがインストールされている場合
choco install golang
```

### 方法3: Scoopを使用

```powershell
# Scoopがインストールされている場合
scoop install go
```

### インストール確認

PowerShellを再起動後、以下で確認：

```powershell
go version
```

正常にインストールされていれば、バージョン情報が表示されます。

## 🚀 プロジェクトのセットアップ

### 1. 依存関係のインストール

```powershell
# プロジェクトディレクトリに移動
cd examples\codex_bot

# 依存関係をダウンロード
go mod download
```

### 2. 環境変数の設定

PowerShellで環境変数を設定：

```powershell
# セッション内でのみ有効（一時的）
$env:LINE_CHANNEL_SECRET = "your_channel_secret"
$env:LINE_CHANNEL_ACCESS_TOKEN = "your_channel_access_token"
$env:OPENAI_API_KEY = "sk-your-openai-api-key"
$env:PORT = "8080"
```

または、永続的に設定する場合：

```powershell
# システム環境変数に追加（管理者権限が必要）
[System.Environment]::SetEnvironmentVariable("LINE_CHANNEL_SECRET", "your_secret", "User")
[System.Environment]::SetEnvironmentVariable("LINE_CHANNEL_ACCESS_TOKEN", "your_token", "User")
[System.Environment]::SetEnvironmentVariable("OPENAI_API_KEY", "sk-your-key", "User")
```

### 3. サーバーの起動

#### 方法A: go runを使用

```powershell
# すべての.goファイルを実行
go run main.go webhook_template.go
```

または、ワイルドカードを使用（PowerShellの場合）：

```powershell
Get-ChildItem *.go | ForEach-Object { go run $_.Name }
```

#### 方法B: ビルドして実行

```powershell
# ビルド
go build -o codex-bot.exe

# 実行
.\codex-bot.exe
```

## 📝 環境変数ファイルの使用（推奨）

### godotenvを使用する方法

1. godotenvをインストール：

```powershell
go get github.com/joho/godotenv
```

2. `main.go`に以下を追加：

```go
import _ "github.com/joho/godotenv/autoload"
```

3. `.env`ファイルを作成（`env.template`をコピー）：

```powershell
Copy-Item env.template .env
```

4. `.env`ファイルを編集して実際の値を設定

5. サーバーを起動：

```powershell
go run main.go webhook_template.go
```

## 🐛 トラブルシューティング

### Goが見つからない

1. **PATH環境変数を確認**：
   ```powershell
   $env:PATH -split ';' | Select-String "go"
   ```

2. **Goのインストールパスを確認**：
   通常は `C:\Program Files\Go\bin` または `C:\Go\bin`

3. **PATHに追加**（必要に応じて）：
   ```powershell
   $env:PATH += ";C:\Program Files\Go\bin"
   ```

### モジュールが見つからない

```powershell
# go.modを初期化（プロジェクトルートで）
cd ..\..
go mod tidy
```

### ポートが使用中

```powershell
# ポート8080を使用しているプロセスを確認
netstat -ano | findstr :8080

# プロセスを終了（PIDを確認してから）
taskkill /PID <PID番号> /F
```

## 💡 便利なコマンド

### 環境変数の確認

```powershell
# すべての環境変数を表示
Get-ChildItem Env:

# 特定の環境変数を確認
$env:LINE_CHANNEL_SECRET
$env:OPENAI_API_KEY
```

### プロセスの確認

```powershell
# Goプロセスを確認
Get-Process | Where-Object {$_.ProcessName -like "*go*"}
```

### ログの確認

サーバーを起動すると、以下のようなログが表示されます：

```
🚀 Codex Bot Server starting...
📝 Webhook URL: http://localhost:8080/webhook
💡 Health check: http://localhost:8080/health
✅ Server is ready to receive webhook events
```

## 🔄 開発ワークフロー

### 1. コードの変更

エディタでコードを編集

### 2. サーバーの再起動

```powershell
# Ctrl+Cで停止してから
go run main.go webhook_template.go
```

### 3. テスト

ブラウザで `http://localhost:8080/health` にアクセスして確認

## 🔄 Windows起動時の自動起動設定

Windows起動時に自動でCodex Botサーバーを起動する方法：

### 方法1: タスクスケジューラー（推奨）

```powershell
# 管理者としてPowerShellを実行
cd examples\codex_bot
.\install_service.ps1
```

この方法は：
- ✅ バックグラウンドで実行
- ✅ エラー時に自動再起動（最大3回）
- ✅ システム環境変数を使用
- ✅ 最も安定した方法

### 方法2: スタートアップフォルダ（簡単）

```powershell
.\install_startup.ps1
```

この方法は：
- ✅ 簡単に設定可能
- ✅ 管理者権限不要
- ⚠️ ログイン時にコマンドプロンプトが表示される

### 方法3: バックグラウンド実行

```powershell
.\install_service_background.ps1
```

この方法は：
- ✅ バックグラウンドで実行（ウィンドウ非表示）
- ✅ ログファイルに出力
- ✅ 管理者権限不要

### 自動起動の削除

```powershell
# タスクスケジューラー
.\install_service.ps1 -Uninstall

# スタートアップフォルダ
.\install_startup.ps1 -Uninstall

# バックグラウンド実行
.\install_service_background.ps1 -Uninstall
```

## 📚 参考リンク

- [Go公式ドキュメント](https://go.dev/doc/)
- [WindowsでのGoインストール](https://go.dev/doc/install)
- [PowerShell環境変数設定](https://docs.microsoft.com/ja-jp/powershell/module/microsoft.powershell.core/about/about_environment_variables)
- [Windowsタスクスケジューラー](https://docs.microsoft.com/ja-jp/windows/win32/taskschd/task-scheduler-start-page)

---

**問題が解決しない場合は、GitHubのIssuesで報告してください！**
