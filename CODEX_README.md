# zapabob/Codex - AIコーディングアシスタント for LINE Bot

zapabob/Codex は LINE Bot SDK for Go の拡張機能で、OpenAI Codex や類似の AI モデルを活用したコーディングアシスタント機能を提供します。LINE Bot との連携により、自然言語でのコード生成・レビュー・修正を実現します。

## 特徴

- 🤖 **AI 駆動のコード生成・修正**: OpenAI Codex などの AI モデルを活用
- 💬 **LINE Bot 統合**: LINE メッセージで直接コード支援が可能
- 🔒 **型安全**: Go 言語の強力な型システムを活用した実装
- 🧪 **包括的なテスト**: ユニットテストと統合テストで品質保証
- 📚 **豊富な言語対応**: Go, Python, JavaScript, TypeScript, Java, Rust, C/C++, SQL など

## 対応モード

### 1. コード生成 (Generate)
新しいコードを自然言語の説明から生成します。

```
/generate go Hello World を表示する関数
```

### 2. コードレビュー (Review)
既存コードの品質を分析し、改善点を提案します。

```
/review python
def hello():
    print("Hello")

hello()
```

### 3. バグ修正 (Fix)
コードのバグを検出し、修正案を提示します。

```
/fix javascript
function broken() {
    console.log("fix me"
}
```

### 4. コード説明 (Explain)
コードの機能を自然言語で説明します。

```
/explain java
public class Hello {
    public static void main(String[] args) {
        System.out.println("Hello");
    }
}
```

### 5. リファクタリング (Refactor)
コードの構造を改善し、可読性・保守性を向上させます。

```
/refactor rust
fn main() {
    let x = 5;
    let y = 10;
    println!("{}", x + y);
}
```

## インストールとセットアップ

### 1. 依存関係

```go
import (
    "github.com/line/line-bot-sdk-go/v8/linebot"
    // Codex 機能を使用する場合、別途 AI サービス SDK を追加
)
```

### 2. 設定

zapabob/Codexは以下の3つの認証方法をサポートしています：

#### APIキー認証（推奨: 開発・テスト用）

```go
config := linebot.CodexConfig{
    APIKey:     "sk-your-openai-api-key", // OpenAI APIキー
    Model:      "gpt-5-codex", // 最新モデル: gpt-5-codex, gpt-4o, gpt-4-turbo
    BaseURL:    "https://api.openai.com/v1",
    Timeout:    60 * time.Second, // GPT-5-Codexは処理時間が長い場合がある
    MaxRetries: 3,

    // 最新APIオプション
    UseResponsesAPI: true, // GPT-5-CodexではResponses APIを使用（推奨）
    DefaultOptions: linebot.CodexOptions{
        MaxTokens:        4000,  // GPT-5-Codexの最大トークン数に対応
        Temperature:      0.1,   // コード生成時は低い温度が適切
        TopP:            0.9,
        FrequencyPenalty: 0.0,   // 新しいパラメータ
        PresencePenalty:  0.0,   // 新しいパラメータ
        IncludeTests:     true,
        IncludeComments:  true,
        Seed:            42,     // 再現性のためにシードを設定
    },
}
```

#### ChatGPT Platform OAuth 2.0 認証（推奨: 本番運用）

```go
// ChatGPT Platformを使用したOAuth 2.0認証
config := linebot.CodexConfig{
    // OAuth 2.0設定
    UseOAuth:    true,
    ClientID:    "your-chatgpt-client-id",     // ChatGPT Platformから取得
    ClientSecret: "your-chatgpt-client-secret", // ChatGPT Platformから取得
    RedirectURL: "https://yourdomain.com/oauth/callback",

    // API設定
    Model:      "gpt-5-codex",
    BaseURL:    "https://api.openai.com/v1",
    Timeout:    60 * time.Second,
    MaxRetries: 3,

    // OAuthスコープ
    Scopes: []string{
        "openid",
        "profile",
        "email",
        "model.request", // ChatGPT Platform特有のスコープ
    },

    // 高度なオプション
    UseResponsesAPI: true,
    DefaultOptions: linebot.CodexOptions{
        MaxTokens:        4000,
        Temperature:      0.1,
        TopP:            0.9,
        FrequencyPenalty: 0.0,
        PresencePenalty:  0.0,
        IncludeTests:     true,
        IncludeComments:  true,
        Seed:            42,
    },
}
```

#### Azure OpenAI Service

```go
config := linebot.CodexConfig{
    APIKey:     "your-azure-api-key",
    BaseURL:    "https://your-resource.openai.azure.com/",
    APIVersion: "2024-02-15-preview", // 最新APIバージョン
    Model:      "gpt-5-codex",
    Timeout:    60 * time.Second,
    MaxRetries: 3,

    UseResponsesAPI: true,
    DefaultOptions: linebot.CodexOptions{
        MaxTokens:        4000,
        Temperature:      0.1,
        TopP:            0.9,
        FrequencyPenalty: 0.0,
        PresencePenalty:  0.0,
        IncludeTests:     true,
        IncludeComments:  true,
        Seed:            42,
    },
}
```

#### ChatGPTアカウントログイン（プログラム内認証）

OAuth 2.0を使用する場合、以下の手順で認証を行います：

```go
// 1. 認可URLを生成
authURL, err := codexHandler.GetAuthorizationURL("your-state")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("ブラウザで以下のURLにアクセスして認証を行ってください:\n%s\n", authURL)

// 2. コールバックで認可コードを受け取り、トークンと交換
token, err := codexHandler.ExchangeCodeForToken(ctx, "authorization-code")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("認証成功！ トークン有効期限: %v\n", token.ExpiresAt)

// 3. トークンは自動的に管理され、必要に応じてリフレッシュされます
```

### 3. 初期化

```go
// LINE Bot クライアント作成
bot, err := linebot.New("channel-secret", "channel-access-token")
if err != nil {
    log.Fatal(err)
}

// Codex Webhook ハンドラー作成
codexHandler, err := linebot.NewCodexWebhookHandler(config, bot)
if err != nil {
    log.Fatal(err)
}

// HTTP ハンドラー取得
webhookHandler, err := codexHandler.GetWebhookHandler("channel-secret")
if err != nil {
    log.Fatal(err)
}

// HTTP サーバー起動
http.Handle("/webhook", webhookHandler)
log.Println("Server starting on :8080")
log.Fatal(http.ListenAndServe(":8080", nil))
```

## 使用例

### 基本的なコード生成

```go
// ユーザーが LINE で送信:
// /generate go フィボナッチ数列を計算する関数

// 応答例:
// ✅ Codex処理完了
//
// 📝 説明:
// フィボナッチ数列を計算するGo関数を生成しました。
//
// 💻 コード:
// ```go
// package main
//
// import "fmt"
//
// // fibonacci calculates the nth Fibonacci number
// func fibonacci(n int) int {
//     if n <= 1 {
//         return n
//     }
//     return fibonacci(n-1) + fibonacci(n-2)
// }
//
// func main() {
//     fmt.Println(fibonacci(10)) // Output: 55
// }
// ```
//
// 💡 提案:
// 1. 大きなnに対してはメモ化を検討してください
// 2. テストケースを追加することをおすすめします
//
// ⏱️ 処理時間: 2.5s
// 🤖 モデル: gpt-4
// 🎫 トークン使用: 450
```

### コードレビュー

```go
// ユーザーが LINE で送信:
// /review python
// def calculate_average(numbers):
//     total = 0
//     for num in numbers:
//         total += num
//     return total / len(numbers)
//
// print(calculate_average([1, 2, 3, 4, 5]))

//
// 応答例:
// ✅ Codex処理完了
//
// 📝 説明:
// このPythonコードは数値リストの平均を計算します。
//
// 💻 コード:
// def calculate_average(numbers):
//     total = 0
//     for num in numbers:
//         total += num
//     return total / len(numbers)
//
// print(calculate_average([1, 2, 3, 4, 5]))
//
// 💡 提案:
// 1. 空のリストが渡された場合のエラーハンドリングを追加してください
// 2. sum() 組み込み関数を使用するとより簡潔になります
// 3. 型ヒントを追加することを検討してください
//
// ⏱️ 処理時間: 1.8s
// 🤖 モデル: gpt-4
// 🎫 トークン使用: 320
```

## API リファレンス

### CodexConfig

Codex の設定を定義する構造体。

```go
type CodexConfig struct {
    APIKey         string          // AI サービスの API キー
    Model          string          // 使用する AI モデル
    BaseURL        string          // API のベース URL
    Timeout        time.Duration   // リクエストタイムアウト
    MaxRetries     int             // 最大リトライ回数
    DefaultOptions CodexOptions    // デフォルトオプション
}
```

### CodexRequest

Codex へのリクエストを定義する構造体。

```go
type CodexRequest struct {
    Mode     CodexMode      // 操作モード
    Language CodexLanguage  // プログラミング言語
    Code     string         // 入力コード（レビュー・修正モード用）
    Prompt   string         // 自然言語プロンプト（生成モード用）
    Context  string         // 追加コンテキスト
    Options  CodexOptions   // 追加オプション
}
```

### CodexResponse

Codex からのレスポンスを定義する構造体。

```go
type CodexResponse struct {
    Success     bool            // 処理成功フラグ
    Code        string          // 生成・修正されたコード
    Explanation string          // 自然言語での説明
    Suggestions []string        // 改善提案
    Errors      []CodexError    // エラー情報
    Metadata    CodexMetadata   // メタデータ
}
```

## エラーハンドリング

Codex は包括的なエラーハンドリングを提供します：

- **バリデーションエラー**: リクエストパラメータの検証失敗
- **処理エラー**: AI サービスとの通信エラー
- **タイムアウト**: リクエストタイムアウト
- **レート制限**: API レート制限超過

すべてのエラーは構造化された `CodexError` 型で返され、エラータイプと詳細なメッセージを提供します。

## セキュリティ

- API キーは環境変数や安全な設定管理を使用してください
- 入力データの検証を常に実施
- レート制限とタイムアウトを適切に設定
- センシティブなコードは処理しないよう注意

## テスト

包括的なテストスイートが提供されます：

```bash
# テスト実行
go test ./linebot -v -run TestCodex

# カバレッジ測定
go test ./linebot -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## パフォーマンス

- **ストリーミングレスポンス**: 大規模コード生成時のメモリ効率
- **並列処理**: 複数のリクエストを並列処理可能
- **キャッシュ**: 頻繁なリクエストのレスポンスキャッシュ
- **タイムアウト管理**: リクエストの適切なタイムアウト設定

## 拡張性

Codex は拡張性を考慮した設計：

- 新しい AI サービス（Claude, Gemini など）の容易な統合
- カスタム言語サポートの追加
- プラグインシステムによる機能拡張
- ミドルウェアによる前処理・後処理

## 貢献

1. Fork して機能ブランチを作成
2. 変更を実装
3. テストを追加
4. すべてのテストが通ることを確認
5. Pull Request を作成

## ライセンス

Apache License 2.0

## サポート

- [Issues](https://github.com/line/line-bot-sdk-go/issues)
- [Documentation](https://developers.line.biz/en/docs/messaging-api/)

---

*この機能は AI によるコード生成を支援するものであり、生成されたコードの正確性・安全性については利用者が責任を持って検証してください。*
