// Copyright 2025 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package linebot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// CodexWebhookHandler handles LINE webhook events for Codex integration
type CodexWebhookHandler struct {
	codexHandler *CodexHandler
	bot          *Client
	replyToken   string
	userID       string
}

// NewCodexWebhookHandler creates a new Codex webhook handler
func NewCodexWebhookHandler(config CodexConfig, bot *Client) (*CodexWebhookHandler, error) {
	codexHandler, err := NewCodexHandler(config, bot)
	if err != nil {
		return nil, fmt.Errorf("failed to create codex handler: %w", err)
	}

	return &CodexWebhookHandler{
		codexHandler: codexHandler,
		bot:          bot,
	}, nil
}

// HandleWebhookEvents processes webhook events for Codex functionality
func (h *CodexWebhookHandler) HandleWebhookEvents(events []webhook.EventInterface) {
	for _, eventInterface := range events {
		event, err := h.convertWebhookEvent(eventInterface)
		if err != nil {
			log.Printf("Error converting webhook event: %v", err)
			continue
		}
		if err := h.handleEvent(event); err != nil {
			log.Printf("Error handling event: %v", err)
			h.sendErrorMessage(event, err)
		}
	}
}

// convertWebhookEvent converts webhook.EventInterface to linebot.Event
func (h *CodexWebhookHandler) convertWebhookEvent(eventInterface webhook.EventInterface) (*Event, error) {
	// Get event type from webhook event
	var eventType EventType
	var replyToken string
	var timestamp time.Time
	var source *EventSource
	var message Message

	switch e := eventInterface.(type) {
	case *webhook.MessageEvent:
		eventType = EventTypeMessage
		replyToken = e.ReplyToken
		timestamp = time.Unix(e.Timestamp/1000, (e.Timestamp%1000)*int64(time.Millisecond))
		if e.Source != nil {
			source = h.convertSource(e.Source)
		}
		if e.Message != nil {
			switch m := e.Message.(type) {
			case *webhook.TextMessageContent:
				message = NewTextMessage(m.Text)
			}
		}
	case *webhook.FollowEvent:
		eventType = EventTypeFollow
		replyToken = e.ReplyToken
		timestamp = time.Unix(e.Timestamp/1000, (e.Timestamp%1000)*int64(time.Millisecond))
		if e.Source != nil {
			source = h.convertSource(e.Source)
		}
	case *webhook.UnfollowEvent:
		eventType = EventTypeUnfollow
		timestamp = time.Unix(e.Timestamp/1000, (e.Timestamp%1000)*int64(time.Millisecond))
		if e.Source != nil {
			source = h.convertSource(e.Source)
		}
	default:
		return nil, fmt.Errorf("unsupported event type: %T", eventInterface)
	}

	return &Event{
		ReplyToken: replyToken,
		Type:       eventType,
		Timestamp:  timestamp,
		Source:     source,
		Message:    message,
	}, nil
}

// convertSource converts webhook.SourceInterface to linebot.EventSource
func (h *CodexWebhookHandler) convertSource(sourceInterface webhook.SourceInterface) *EventSource {
	source := &EventSource{
		Type: EventSourceType(sourceInterface.GetType()),
	}

	switch s := sourceInterface.(type) {
	case *webhook.UserSource:
		source.UserID = s.UserId
	case *webhook.GroupSource:
		source.GroupID = s.GroupId
		source.UserID = s.UserId
	case *webhook.RoomSource:
		source.RoomID = s.RoomId
		source.UserID = s.UserId
	}

	return source
}

// handleEvent processes a single webhook event
func (h *CodexWebhookHandler) handleEvent(event *Event) error {
	switch event.Type {
	case EventTypeMessage:
		return h.handleMessageEvent(event)
	case EventTypeFollow:
		return h.handleFollowEvent(event)
	case EventTypeUnfollow:
		return h.handleUnfollowEvent(event)
	default:
		// Ignore other event types
		return nil
	}
}

// handleMessageEvent processes message events
func (h *CodexWebhookHandler) handleMessageEvent(event *Event) error {
	if event.Message == nil {
		return nil
	}

	switch message := event.Message.(type) {
	case *TextMessage:
		return h.handleTextMessage(event, message)
	default:
		// Ignore non-text messages for now
		return nil
	}
}

// handleTextMessage processes text messages
func (h *CodexWebhookHandler) handleTextMessage(event *Event, message *TextMessage) error {
	text := strings.TrimSpace(message.Text)

	// Check if this is a Codex command
	if !h.isCodexCommand(text) {
		return nil // Not a Codex command, ignore
	}

	// Parse the command
	req, err := h.parseCodexCommand(text)
	if err != nil {
		return h.sendReply(event.ReplyToken, fmt.Sprintf("コマンド解析エラー: %v\n\n使用例:\n/generate go Hello World関数\n/review go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n/fix python\nprint(\"hello\"", err))
	}

	// Process with Codex
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	response, err := h.codexHandler.ProcessRequest(ctx, req)
	if err != nil {
		return h.sendReply(event.ReplyToken, fmt.Sprintf("Codex処理エラー: %v", err))
	}

	// Format and send response
	return h.sendCodexResponse(event.ReplyToken, response)
}

// isCodexCommand checks if the message is a Codex command
func (h *CodexWebhookHandler) isCodexCommand(text string) bool {
	codexCommands := []string{"/generate", "/review", "/fix", "/explain", "/refactor", "/codex"}
	for _, cmd := range codexCommands {
		if strings.HasPrefix(strings.ToLower(text), cmd) {
			return true
		}
	}
	return false
}

// parseCodexCommand parses a Codex command from text
func (h *CodexWebhookHandler) parseCodexCommand(text string) (CodexRequest, error) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return CodexRequest{}, fmt.Errorf("empty command")
	}

	firstLine := strings.TrimSpace(lines[0])

	// Parse command and language
	re := regexp.MustCompile(`^/(\w+)\s+(\w+)(?:\s+(.+))?`)
	matches := re.FindStringSubmatch(firstLine)
	if len(matches) < 3 {
		return CodexRequest{}, fmt.Errorf("invalid command format")
	}

	mode := CodexMode(matches[1])
	language := CodexLanguage(matches[2])
	promptOrContext := ""
	if len(matches) > 3 {
		promptOrContext = matches[3]
	}

	// Get code from remaining lines
	code := ""
	if len(lines) > 1 {
		code = strings.Join(lines[1:], "\n")
	}

	var req CodexRequest
	req.Language = language
	req.Options = h.codexHandler.config.DefaultOptions

	switch mode {
	case CodexModeGenerate:
		req.Mode = CodexModeGenerate
		if promptOrContext != "" {
			req.Prompt = promptOrContext
		} else {
			req.Prompt = code // fallback to code as prompt
		}
		if code != "" {
			req.Context = code
		}
	case CodexModeReview, CodexModeFix, CodexModeExplain, CodexModeRefactor:
		req.Mode = mode
		req.Code = code
		if promptOrContext != "" {
			req.Context = promptOrContext
		}
	default:
		return CodexRequest{}, fmt.Errorf("unsupported mode: %s", mode)
	}

	return req, nil
}

// sendCodexResponse sends a formatted Codex response
func (h *CodexWebhookHandler) sendCodexResponse(replyToken string, response *CodexResponse) error {
	var message string

	if !response.Success {
		message = "❌ Codex処理に失敗しました:\n"
		for _, err := range response.Errors {
			message += fmt.Sprintf("• %s: %s\n", err.Type, err.Message)
		}
		return h.sendReply(replyToken, message)
	}

	message = "✅ Codex処理完了\n\n"

	if response.Explanation != "" {
		message += fmt.Sprintf("📝 説明:\n%s\n\n", response.Explanation)
	}

	if response.Code != "" {
		message += fmt.Sprintf("💻 コード:\n```\n%s\n```\n\n", response.Code)
	}

	if len(response.Suggestions) > 0 {
		message += "💡 提案:\n"
		for i, suggestion := range response.Suggestions {
			message += fmt.Sprintf("%d. %s\n", i+1, suggestion)
		}
		message += "\n"
	}

	message += fmt.Sprintf("⏱️ 処理時間: %v\n", response.Metadata.ProcessingTime)
	message += fmt.Sprintf("🤖 モデル: %s\n", response.Metadata.Model)
	message += fmt.Sprintf("🎫 トークン使用: %d\n", response.Metadata.TokensUsed)

	// LINE has message length limit, truncate if necessary
	if len(message) > 5000 {
		message = message[:4950] + "\n\n... (メッセージが長すぎるため切り詰めました)"
	}

	return h.sendReply(replyToken, message)
}

// sendReply sends a reply message
func (h *CodexWebhookHandler) sendReply(replyToken, text string) error {
	if _, err := h.bot.ReplyMessage(replyToken, NewTextMessage(text)).Do(); err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}
	return nil
}

// sendErrorMessage sends an error message
func (h *CodexWebhookHandler) sendErrorMessage(event *Event, err error) {
	if event.ReplyToken != "" {
		errorMsg := fmt.Sprintf("🚨 エラーが発生しました:\n%s", err.Error())
		if replyErr := h.sendReply(event.ReplyToken, errorMsg); replyErr != nil {
			log.Printf("Failed to send error message: %v", replyErr)
		}
	}
}

// handleFollowEvent handles follow events
func (h *CodexWebhookHandler) handleFollowEvent(event *Event) error {
	welcomeMsg := `🤖 Codex AIコーディングアシスタントへようこそ！

以下のコマンドを使ってコードを生成・修正できます：

📝 コード生成:
/generate [言語] [説明]
例: /generate go Hello Worldを表示する関数

🔍 コードレビュー:
/review [言語]
[コードをここに貼り付け]

🐛 バグ修正:
/fix [言語]
[修正したいコードをここに貼り付け]

📖 コード説明:
/explain [言語]
[説明したいコードをここに貼り付け]

🔄 コードリファクタリング:
/refactor [言語]
[リファクタリングしたいコードをここに貼り付け]

💡 使い方の詳細はヘルプと送信してください。`

	return h.sendReply(event.ReplyToken, welcomeMsg)
}

// handleUnfollowEvent handles unfollow events
func (h *CodexWebhookHandler) handleUnfollowEvent(event *Event) error {
	// No action needed for unfollow events
	return nil
}

// GetWebhookHandler returns an http.Handler for webhook processing
func (h *CodexWebhookHandler) GetWebhookHandler(channelSecret string) (http.Handler, error) {
	webhookHandler, err := webhook.NewWebhookHandler(channelSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook handler: %w", err)
	}

	webhookHandler.HandleEvents(func(callbackReq *webhook.CallbackRequest, req *http.Request) {
		h.HandleWebhookEvents(callbackReq.Events)
	})

	webhookHandler.HandleError(func(err error, req *http.Request) {
		log.Printf("Webhook error: %v", err)
	})

	return webhookHandler, nil
}


