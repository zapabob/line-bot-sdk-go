# Codex Bot Webhook テンプレート

このテンプレートは、LINE Botとzapabob/Codexを接続する完全なwebhookサーバー実装です。

## 📁 ファイル構成

- `main.go` - エントリーポイント（シンプルな起動コード）
- `webhook_template.go` - Webhookサーバーの完全な実装
- `env.template` - 環境変数のテンプレート
- `README.md` - このファイル

## 🚀 クイックスタート

### 0. Go言語のインストール確認

**Goがインストールされていない場合:**

```powershell
# エラーメッセージが表示された場合
# 以下のいずれかの方法でGoをインストールしてください
```

**インストール方法:**

1. **公式インストーラー（推奨）**:
   - https://go.dev/dl/ にアクセス
   - Windows用の`.msi`ファイルをダウンロード
   - インストーラーを実行
   - PowerShellを再起動

2. **Chocolateyを使用**:
   ```powershell
   choco install golang -y
   ```

3. **詳細**: `setup_guide.txt` を参照

**インストール確認:**
```powershell
go version
```

### 1. 環境変数の設定

`env.template`をコピーして`.env`ファイルを作成：

```bash
cp env.template .env
```

`.env`ファイルを編集して、実際の値を設定：

```bash
# 必須設定
LINE_CHANNEL_SECRET=your_actual_channel_secret
LINE_CHANNEL_ACCESS_TOKEN=your_actual_channel_access_token
OPENAI_API_KEY=sk-your-actual-openai-api-key
```

### 2. 環境変数の読み込み（オプション）

godotenvを使用する場合：

```bash
go get github.com/joho/godotenv
```

`main.go`に以下を追加：

```go
import _ "github.com/joho/godotenv/autoload"
```

または、手動で環境変数を設定：

```bash
export LINE_CHANNEL_SECRET="your_secret"
export LINE_CHANNEL_ACCESS_TOKEN="your_token"
export OPENAI_API_KEY="sk-your-key"
```

### 3. サーバーの起動

#### 方法A: PowerShellで実行

```powershell
go run main.go webhook_template.go
```

#### 方法B: バッチファイルで実行（簡単）

```cmd
start_server.bat
```

このバッチファイルは以下を自動で確認します：
- Go言語のインストール状況
- 環境変数の設定状況
- エラーメッセージの表示

#### 方法C: ビルドして実行

```powershell
# ビルド
go build -o codex-bot.exe

# 実行
.\codex-bot.exe
```

### 4. Webhook URLの設定

LINE Developers コンソールで、Webhook URLを設定：

1. [LINE Developers](https://developers.line.biz/ja/) にログイン
2. チャネルを選択
3. 「Messaging API」タブを開く
4. 「Webhook URL」に以下を入力：
   - ローカル開発: `https://your-ngrok-url.ngrok.io/webhook`
   - 本番環境: `https://your-domain.com/webhook`
5. 「検証」ボタンをクリック
6. 「Webhookの利用」を有効化

## ✨ 機能

### 基本機能

- ✅ LINE Bot Webhook受信
- ✅ Codex AIコーディングアシスタント統合
- ✅ グレースフルシャットダウン
- ✅ ヘルスチェックエンドポイント
- ✅ 詳細なエラーハンドリング

### 高度な機能

- ✅ MCPサーバー統合（GeminiCLI、ClaudeCode）
- ✅ OAuth 2.0認証サポート（ChatGPT Platform）
- ✅ 複数AIプロバイダー対応
- ✅ 環境変数ベースの設定

## 📱 使い方

LINEアプリで以下のコマンドを送信：

### コード生成
```
/generate go Hello World関数
```

### コードレビュー
```
/review python
def hello():
    print("Hello")
```

### バグ修正
```
/fix javascript
function broken() {
    console.log("fix me"
}
```

### コード説明
```
/explain java
public class Hello {
    public static void main(String[] args) {
        System.out.println("Hello");
    }
}
```

### リファクタリング
```
/refactor rust
fn main() {
    let x = 5;
    let y = 10;
    println!("{}", x + y);
}
```

## 🔧 設定オプション

### 環境変数

| 変数名 | 必須 | デフォルト | 説明 |
|--------|------|-----------|------|
| `LINE_CHANNEL_SECRET` | ✅ | - | LINEチャネルシークレット |
| `LINE_CHANNEL_ACCESS_TOKEN` | ✅ | - | LINEチャネルアクセストークン |
| `OPENAI_API_KEY` | ✅ | - | OpenAI APIキー |
| `OPENAI_MODEL` | ❌ | `gpt-5-codex` | 使用するAIモデル |
| `PORT` | ❌ | `8080` | サーバーポート |
| `USE_RESPONSES_API` | ❌ | `true` | Responses API使用フラグ |
| `GEMINI_MCP_ENABLED` | ❌ | `false` | Gemini CLI MCP有効化 |
| `CLAUDE_MCP_ENABLED` | ❌ | `false` | Claude Code MCP有効化 |

### MCP統合の有効化

```bash
# Gemini CLI MCPを有効化
export GEMINI_MCP_ENABLED=true

# Claude Code MCPを有効化
export CLAUDE_MCP_ENABLED=true
export ANTHROPIC_API_KEY=your-anthropic-api-key
```

## 🏗️ アーキテクチャ

```
main.go
  └─> NewWebhookServer()
      ├─> LINE Bot Client作成
      ├─> Codex Handler作成
      ├─> MCPプロバイダー登録（オプション）
      └─> Webhook Handler作成
          └─> HTTP Server起動
              ├─> POST /webhook - LINE webhook受信
              ├─> GET  /health  - ヘルスチェック
              └─> GET  /        - ルート情報
```

## 🐛 トラブルシューティング

### サーバーが起動しない

1. 環境変数が正しく設定されているか確認：
   ```bash
   echo $LINE_CHANNEL_SECRET
   echo $LINE_CHANNEL_ACCESS_TOKEN
   echo $OPENAI_API_KEY
   ```

2. ポートが使用されていないか確認：
   ```bash
   # Windows
   netstat -ano | findstr :8080
   
   # Linux/Mac
   lsof -i :8080
   ```

### Webhookが応答しない

1. サーバーログを確認
2. LINE Developers コンソールで「Webhook URL」の「検証」を実行
3. ngrokが正しく動作しているか確認（ローカル開発時）

### MCPプロバイダーが動作しない

1. CLIがインストールされているか確認：
   ```bash
   which gemini
   which claude
   # または
   npx @google/gemini-cli --version
   npx @anthropic/claude-code --version
   ```

2. 環境変数が正しく設定されているか確認
3. サーバーログでエラーメッセージを確認

## 📚 詳細ドキュメント

- [CODEX_SETUP_GUIDE.md](../../CODEX_SETUP_GUIDE.md) - 完全なセットアップガイド
- [CODEX_README.md](../../CODEX_README.md) - APIリファレンス
- [実装ログ](../../_docs/2025-11-16_zapabob-codex.md) - 実装の詳細

## 🔄 Windows起動時の自動起動設定

Windows起動時に自動でサーバーを起動する方法が3つあります：

### 方法1: タスクスケジューラー（推奨・管理者権限必要）

```powershell
# 管理者としてPowerShellを実行
.\install_service.ps1
```

**特徴:**
- ✅ バックグラウンドで実行
- ✅ エラー時に自動再起動
- ✅ システム環境変数を使用
- ✅ 最も安定した方法

**削除:**
```powershell
.\install_service.ps1 -Uninstall
```

### 方法2: スタートアップフォルダ（簡単・管理者権限不要）

```powershell
.\install_startup.ps1
```

**特徴:**
- ✅ 簡単に設定可能
- ✅ 管理者権限不要
- ⚠️ ログイン時にコマンドプロンプトが表示される

**削除:**
```powershell
.\install_startup.ps1 -Uninstall
```

### 方法3: バックグラウンド実行（管理者権限不要）

```powershell
.\install_service_background.ps1
```

**特徴:**
- ✅ バックグラウンドで実行（ウィンドウ非表示）
- ✅ ログファイルに出力
- ✅ 管理者権限不要

**削除:**
```powershell
.\install_service_background.ps1 -Uninstall
```

### ログの確認

バックグラウンド実行の場合、ログは以下に保存されます：

```
logs/codex_bot_YYYYMMDD.log
```

## 💡 ヒント

- **開発時**: ngrokを使ってローカルでテスト
- **本番環境**: HTTPS必須、適切なセキュリティ設定
- **ログ**: サーバーログを適切に監視
- **コスト**: OpenAI APIの使用量を監視
- **自動起動**: タスクスケジューラーを使用することを推奨

---

**質問や問題がある場合は、GitHubのIssuesで報告してください！**
