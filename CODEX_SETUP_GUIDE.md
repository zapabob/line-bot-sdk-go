# LINE Bot と Codex 接続ガイド

このガイドでは、自分のLINEアカウントとzapabob/Codexを接続して、LINE BotとしてAIコーディングアシスタントを使えるようにする手順を説明します。

## 📋 必要なもの

1. LINEアカウント
2. LINE Developersアカウント（無料）
3. OpenAI APIキー または ChatGPT Platform OAuth設定
4. Go 1.19以上
5. 公開可能なサーバー（ローカル開発の場合はngrok等）

## 🚀 ステップ1: LINE Bot の作成

### 1.1 LINE Developers に登録

1. [LINE Developers](https://developers.line.biz/ja/) にアクセス
2. LINEアカウントでログイン
3. 「新規プロバイダー作成」をクリック
4. プロバイダー名を入力（例: "My Codex Bot"）

### 1.2 チャネル作成

1. 「チャネル」タブをクリック
2. 「Messaging API」を選択
3. チャネル名を入力（例: "Codex Assistant"）
4. 利用規約に同意して作成

### 1.3 チャネル情報の取得

作成後、以下の情報をメモしてください：

- **Channel Secret**: チャネル基本設定 > チャネルシークレット
- **Channel Access Token**: Messaging API > チャネルアクセストークン（発行）

## 🔧 ステップ2: サーバーコードの準備

### 2.1 プロジェクトのセットアップ

```bash
# プロジェクトディレクトリに移動
cd line-bot-sdk-go

# 依存関係のインストール
go mod download
```

### 2.2 サーバーコードの作成

`examples/codex_bot/main.go` を作成：

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot"
)

func main() {
	// 環境変数から設定を読み込み
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")

	if channelSecret == "" || channelToken == "" {
		log.Fatal("LINE_CHANNEL_SECRET and LINE_CHANNEL_ACCESS_TOKEN must be set")
	}

	// LINE Bot クライアント作成
	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		log.Fatal(err)
	}

	// Codex設定
	codexConfig := linebot.CodexConfig{
		APIKey:  openaiAPIKey,
		Model:   "gpt-5-codex", // または "gpt-4o"
		BaseURL: "https://api.openai.com/v1",
		Timeout: 60 * time.Second,
		DefaultOptions: linebot.CodexOptions{
			MaxTokens:    4000,
			Temperature: 0.1,
			IncludeTests: true,
			IncludeComments: true,
		},
		UseResponsesAPI: true,
	}

	// Codex Webhook ハンドラー作成
	codexHandler, err := linebot.NewCodexWebhookHandler(codexConfig, bot)
	if err != nil {
		log.Fatal(err)
	}

	// Webhook ハンドラー取得
	webhookHandler, err := codexHandler.GetWebhookHandler(channelSecret)
	if err != nil {
		log.Fatal(err)
	}

	// HTTP サーバー起動
	http.Handle("/webhook", webhookHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### 2.3 環境変数の設定

`.env` ファイルを作成（本番環境では環境変数として設定）：

```bash
LINE_CHANNEL_SECRET=your_channel_secret_here
LINE_CHANNEL_ACCESS_TOKEN=your_channel_access_token_here
OPENAI_API_KEY=sk-your-openai-api-key-here
PORT=8080
```

## 🌐 ステップ3: サーバーの公開

### オプションA: ローカル開発（ngrok使用）

1. [ngrok](https://ngrok.com/) をインストール
2. ngrokを起動：

```bash
ngrok http 8080
```

3. 表示されたURL（例: `https://abc123.ngrok.io`）をコピー

### オプションB: クラウドデプロイ

- **Heroku**: `heroku create` → `git push heroku main`
- **Railway**: [Railway.app](https://railway.app/) でデプロイ
- **Fly.io**: `fly launch` でデプロイ
- **AWS/GCP/Azure**: 各クラウドのドキュメントを参照

## 🔗 ステップ4: Webhook URLの設定

### 4.1 LINE Developers でWebhook設定

1. LINE Developers コンソールに戻る
2. 作成したチャネルを選択
3. 「Messaging API」タブを開く
4. 「Webhook URL」に以下を入力：
   - ローカル開発: `https://your-ngrok-url.ngrok.io/webhook`
   - 本番環境: `https://your-domain.com/webhook`
5. 「検証」ボタンをクリックして接続確認
6. 「Webhookの利用」を有効化

### 4.2 Webhookイベントの設定

「応答メッセージ」を無効化（Codexが応答するため）：

1. 「応答メッセージ」セクション
2. 「応答メッセージ」を無効化

## 🎯 ステップ5: サーバーの起動

### 5.1 ローカルで起動

```bash
# 環境変数を読み込んで起動
export LINE_CHANNEL_SECRET="your_secret"
export LINE_CHANNEL_ACCESS_TOKEN="your_token"
export OPENAI_API_KEY="your_key"

# サーバー起動
go run examples/codex_bot/main.go
```

または、`.env`ファイルを使う場合：

```bash
# godotenv等を使用
go get github.com/joho/godotenv
```

### 5.2 ビルドして実行

```bash
# ビルド
go build -o codex-bot examples/codex_bot/main.go

# 実行
./codex-bot
```

## ✅ ステップ6: 動作確認

### 6.1 LINEで友だち追加

1. LINE Developers コンソールで「QRコード」を表示
2. スマートフォンのLINEアプリでQRコードをスキャン
3. 友だち追加

### 6.2 テストメッセージ送信

LINEアプリで以下のコマンドを試してください：

```
/generate go Hello World関数
```

または：

```
/review python
def hello():
    print("Hello")
```

## 📱 使い方

### 基本的なコマンド

1. **コード生成**:
   ```
   /generate [言語] [説明]
   例: /generate go フィボナッチ数列を計算する関数
   ```

2. **コードレビュー**:
   ```
   /review [言語]
   [コードをここに貼り付け]
   ```

3. **バグ修正**:
   ```
   /fix [言語]
   [修正したいコード]
   ```

4. **コード説明**:
   ```
   /explain [言語]
   [説明したいコード]
   ```

5. **リファクタリング**:
   ```
   /refactor [言語]
   [リファクタリングしたいコード]
   ```

## 🔐 セキュリティ設定（本番環境）

### 環境変数の安全な管理

- **Heroku**: `heroku config:set KEY=value`
- **Railway**: ダッシュボードで設定
- **AWS**: Secrets Manager または Parameter Store
- **GCP**: Secret Manager
- **Azure**: Key Vault

### Webhook署名検証

Codexは自動的にWebhook署名を検証します。`Channel Secret`が正しく設定されていれば安全です。

## 🐛 トラブルシューティング

### Webhookが応答しない

1. **ログを確認**:
   ```bash
   # サーバーログでエラーを確認
   ```

2. **Webhook URLを再確認**:
   - HTTPSであること
   - `/webhook` パスが正しいこと
   - サーバーが起動していること

3. **LINE Developers で検証**:
   - 「Webhook URL」の「検証」ボタンをクリック
   - 成功メッセージが表示されることを確認

### Codexが応答しない

1. **APIキーを確認**:
   ```bash
   echo $OPENAI_API_KEY
   ```

2. **API制限を確認**:
   - OpenAIの利用制限に達していないか
   - クレジット残高を確認

3. **ログでエラー確認**:
   - サーバーログで詳細なエラーメッセージを確認

### 認証エラー

1. **Channel Secret/Tokenを再確認**:
   - LINE Developers コンソールで再発行
   - 環境変数が正しく設定されているか確認

## 🚀 高度な設定

### OAuth 2.0認証を使用（ChatGPT Platform）

```go
codexConfig := linebot.CodexConfig{
    UseOAuth:    true,
    ClientID:    os.Getenv("CHATGPT_CLIENT_ID"),
    ClientSecret: os.Getenv("CHATGPT_CLIENT_SECRET"),
    RedirectURL: "https://your-domain.com/oauth/callback",
    Scopes:      []string{"openid", "profile", "email", "model.request"},
    Model:       "gpt-5-codex",
    UseResponsesAPI: true,
}
```

### MCPサーバー経由で複数AIプロバイダーを使用（推奨）

MCP（Model Context Protocol）サーバー経由でGeminiCLIとClaudeCodeを接続できます：

```go
// Codex Handler作成
codexHandler, err := linebot.NewCodexHandler(codexConfig, bot)
if err != nil {
    log.Fatal(err)
}

// Gemini CLI MCP設定
geminiConfig := &linebot.GeminiCLIMCPConfig{
    Command: "", // 自動検出（gemini CLIまたはnpx経由）
    Model:   "gemini-2.0-flash",
    Enabled: true,
    Timeout: 60 * time.Second,
}

// Claude Code MCP設定
claudeConfig := &linebot.ClaudeCodeMCPConfig{
    Command: "", // 自動検出（claude CLIまたはnpx経由）
    APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
    Model:   "claude-3-5-sonnet-20241022",
    Enabled: true,
    Timeout: 60 * time.Second,
}

// MCPプロバイダーを登録
ctx := context.Background()
if err := codexHandler.RegisterMCPProviders(ctx, geminiConfig, claudeConfig); err != nil {
    log.Printf("Warning: Failed to register MCP providers: %v", err)
}
```

#### MCPサーバーのインストール

**Gemini CLI:**
```bash
# npm経由でインストール
npm install -g @google/gemini-cli

# またはnpx経由で使用（自動的に使用されます）
```

**Claude Code:**
```bash
# npm経由でインストール
npm install -g @anthropic/claude-code

# またはnpx経由で使用（自動的に使用されます）
```

#### 環境変数設定

```bash
# Gemini MCPを有効化
export GEMINI_MCP_ENABLED=true

# Claude MCPを有効化
export CLAUDE_MCP_ENABLED=true
export ANTHROPIC_API_KEY=your-anthropic-api-key
```

### 直接API経由で複数AIプロバイダーを使用

```go
// プロバイダーマネージャーに登録
providerManager := codexHandler.providerManager

// OpenAIプロバイダーを登録
openaiProvider := NewOpenAIProvider(AIProviderConfig{
    Type:   ProviderOpenAI,
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-5-codex",
    Enabled: true,
})
providerManager.RegisterProvider(openaiProvider)

// Claudeプロバイダーを登録
claudeProvider := NewAnthropicProvider(AIProviderConfig{
    Type:   ProviderAnthropic,
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    Model:  "claude-3-5-sonnet",
    Enabled: true,
})
providerManager.RegisterProvider(claudeProvider)
```

## 📚 次のステップ

- [CODEX_README.md](./CODEX_README.md) - 詳細なAPIリファレンス
- [実装ログ](./_docs/2025-11-16_zapabob-codex.md) - 実装の詳細
- [LINE Developers ドキュメント](https://developers.line.biz/ja/docs/messaging-api/)

## 💡 ヒント

- **開発時**: ngrokを使ってローカルでテスト
- **本番環境**: HTTPS必須、適切なセキュリティ設定
- **コスト管理**: OpenAI APIの使用量を監視
- **エラーハンドリング**: ログを適切に設定

---

**質問や問題がある場合は、GitHubのIssuesで報告してください！**
