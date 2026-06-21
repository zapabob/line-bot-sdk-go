# Codex Bot クイックスタートガイド

Windows環境でCodex Botを最短で起動する手順です。

## ⚡ 5分で起動

### ステップ1: Go言語のインストール確認

```powershell
# PowerShellで実行
.\check_go.ps1
```

Goがインストールされていない場合は、[Go公式サイト](https://go.dev/dl/)からインストールしてください。

### ステップ2: 環境変数の設定

```powershell
# 必須の環境変数を設定
$env:LINE_CHANNEL_SECRET = "your_channel_secret"
$env:LINE_CHANNEL_ACCESS_TOKEN = "your_channel_access_token"
$env:OPENAI_API_KEY = "sk-your-openai-api-key"
```

### ステップ3: サーバーの起動

```powershell
go run main.go webhook_template.go
```

### ステップ4: Webhook URLの設定

1. [LINE Developers](https://developers.line.biz/ja/) にログイン
2. チャネルを選択
3. 「Messaging API」タブ > 「Webhook URL」に設定：
   - ローカル: `https://your-ngrok-url.ngrok.io/webhook`
   - 本番: `https://your-domain.com/webhook`
4. 「検証」をクリック
5. 「Webhookの利用」を有効化

## 🔄 自動起動の設定（オプション）

### 最も簡単な方法（管理者権限不要）

```powershell
.\install_service_background.ps1
```

これで、Windows起動時に自動でサーバーが起動します。

### より安定した方法（管理者権限必要）

```powershell
# 管理者としてPowerShellを実行
.\install_service.ps1
```

## 📱 使い方

LINEアプリで友だち追加後、以下のコマンドを送信：

```
/generate go Hello World関数
```

## 🐛 問題が発生した場合

1. **Goが見つからない**: `check_go.ps1`を実行して確認
2. **環境変数エラー**: 環境変数が正しく設定されているか確認
3. **ポートエラー**: 別のポートを使用（`$env:PORT = "8081"`）

詳細は [WINDOWS_SETUP.md](./WINDOWS_SETUP.md) を参照してください。

---

**それでは、Codex Botをお楽しみください！** 🚀
