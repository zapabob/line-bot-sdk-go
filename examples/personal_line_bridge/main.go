package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type replyRequest struct {
	Message struct {
		Text          string      `json:"text"`
		From          interface{} `json:"from"`
		To            string      `json:"to"`
		IsGroup       bool        `json:"isGroup"`
		IsReply       bool        `json:"isReply"`
		IsReplyToUs   bool        `json:"isReplyToUs"`
		ReplyTargetID string      `json:"replyTargetId"`
		MID           string      `json:"mid"`
	} `json:"message"`
	Profile interface{} `json:"profile"`
	Trigger string      `json:"trigger"`
}

type replyResponse struct {
	Text string `json:"text"`
}

var (
	addr        = flag.String("addr", "127.0.0.1:9102", "bind address")
	hermesCmd   = flag.String("hermes", "hermes", "Hermes executable")
	timeout     = flag.Duration("timeout", 60*time.Second, "Hermes generation timeout")
	maxReplyLen = flag.Int("max-reply-len", 180, "maximum reply length")
	minDelay    = flag.Duration("min-delay", 8*time.Second, "minimum human-like reply delay")
	maxDelay    = flag.Duration("max-delay", 28*time.Second, "maximum human-like reply delay")
	providers   = flag.String("providers", "", "comma-separated Hermes providers for retry/429 rotation")
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(LINE(?:JS)?_PERSONAL_(?:AUTH_TOKEN|PASSWORD|PASS|EMAIL|ADDRESS)=)\S+`),
	regexp.MustCompile(`(?i)\b(?:API_KEY|SECRET|TOKEN|PASSWORD|PASS|AUTH_TOKEN|EMAIL|ADDRESS)\s*[=:]\s*\S+`),
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`),
	regexp.MustCompile(`\b[A-Za-z0-9_\-]{32,}\b`),
}

func sanitize(s string) string {
	if s == "" {
		return s
	}
	s = secretPatterns[0].ReplaceAllString(s, `${1}<redacted>`)
	for _, re := range secretPatterns[1:] {
		s = re.ReplaceAllString(s, "<redacted>")
	}
	return strings.TrimSpace(s)
}

func clipRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

const requiredSignature = " #はくあ #hermesagent"

func ensureRequiredSignature(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "呼んだ？"
	}
	if strings.Contains(s, "#はくあ") && strings.Contains(strings.ToLower(s), "#hermesagent") {
		return clipRunes(s, maxLen)
	}
	if maxLen <= 0 {
		return strings.TrimSpace(s + requiredSignature)
	}
	suffixRunes := []rune(requiredSignature)
	bodyLimit := maxLen - len(suffixRunes)
	if bodyLimit < 1 {
		return clipRunes(strings.TrimSpace(s+requiredSignature), maxLen)
	}
	body := clipRunes(s, bodyLimit)
	return strings.TrimSpace(body + requiredSignature)
}

func cleanHermesOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	dropQueryBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "Warning: Unknown toolsets:") ||
			strings.HasPrefix(trimmed, "⚠️") ||
			strings.HasPrefix(trimmed, "Bitwarden Secrets Manager:") ||
			strings.HasPrefix(lower, "session_id:") ||
			strings.HasPrefix(lower, "query:") {
			dropQueryBlock = strings.HasPrefix(lower, "query:")
			continue
		}
		if dropQueryBlock {
			if strings.Contains(trimmed, "あなたは「はくあ」") ||
				strings.HasPrefix(trimmed, "制約:") ||
				strings.HasPrefix(trimmed, "-") ||
				strings.HasPrefix(trimmed, "トリガー:") ||
				strings.HasPrefix(trimmed, "文脈:") ||
				strings.HasPrefix(trimmed, "サニタイズ済み入力:") {
				continue
			}
			dropQueryBlock = false
		}
		kept = append(kept, trimmed)
	}
	if len(kept) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func humanDelay(ctx context.Context) {
	if *maxDelay <= 0 || *minDelay < 0 || *maxDelay < *minDelay {
		return
	}
	delta := *maxDelay - *minDelay
	d := *minDelay
	if delta > 0 {
		d += time.Duration(rand.Int63n(int64(delta)))
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func providerRotation() []string {
	raw := strings.TrimSpace(*providers)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("HAKUA_REPLY_PROVIDERS"))
	}
	if raw == "" {
		raw = "openai-codex,nvidia,nous,opencode-zen"
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func shouldRotate(stderr string, err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(stderr + " " + err.Error())
	return strings.Contains(low, "429") || strings.Contains(low, "rate limit") || strings.Contains(low, "rate_limit") || strings.Contains(low, "too many requests") || strings.Contains(low, "quota") || strings.Contains(low, "connection") || strings.Contains(low, "timeout")
}

func generateReply(ctx context.Context, req replyRequest) string {
	text := sanitize(req.Message.Text)
	if text == "" {
		humanDelay(ctx)
		return ensureRequiredSignature("呼んだ？短くなら答えるよ。", *maxReplyLen)
	}

	replyContext := "通常のタグ呼び出し"
	if req.Message.IsReplyToUs {
		replyContext = "直前のはくあ発言へのLINEリプライ。タグがなくても会話の続きとして自然に返す"
	} else if req.Message.IsReply {
		replyContext = "LINEリプライ形式だが、はくあ宛てとは確認できない"
	}

	prompt := fmt.Sprintf(`LINEグループで「はくあ」と呼ばれた時の返事を1つだけ作って。
話し方は日本語、友だちっぽく少しギャル、でも内容は正確。最大%d文字、基本1文。
毎回の挨拶、AI/bot/自動返信の自己説明、JSON、箇条書き、前置きは禁止。固定タグは送信直前に付けるので本文に入れなくていい。
秘密っぽい情報やローカル情報は拾わず、わからない時だけ短く聞き返す。

呼ばれ方: %s
会話状況: %s
相手の発言: %s

返事本文だけ:`, *maxReplyLen, sanitize(req.Trigger), replyContext, text)

	var reply string
	lastErr := ""
	for _, provider := range providerRotation() {
		cmdCtx, cancel := context.WithTimeout(ctx, *timeout)
		args := []string{"chat", "--provider", provider, "-Q", "-q", prompt}
		cmd := exec.CommandContext(cmdCtx, *hermesCmd, args...)
		var out bytes.Buffer
		var errOut bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errOut
		err := cmd.Run()
		cancel()
		if err == nil {
			reply = cleanHermesOutput(sanitize(out.String()))
			if strings.TrimSpace(reply) != "" {
				log.Printf("hermes generation provider=%s ok", provider)
				break
			}
		}
		lastErr = sanitize(errOut.String())
		log.Printf("hermes generation provider=%s failed: %v stderr=%s", provider, err, clipRunes(lastErr, 300))
		if !shouldRotate(lastErr, err) {
			break
		}
	}
	if strings.TrimSpace(reply) == "" {
		humanDelay(ctx)
		return ensureRequiredSignature("呼んだ？いま短めに反応するね。", *maxReplyLen)
	}

	reply = cleanHermesOutput(sanitize(reply))
	// Remove common CLI framing noise if any.
	reply = strings.Trim(reply, " 	\r\n`\"")
	if reply == "" {
		reply = "呼んだ？"
	}
	humanDelay(ctx)
	return ensureRequiredSignature(reply, *maxReplyLen)
}

func handleReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req replyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if !req.Message.IsGroup {
		// The linejs worker already filters this; keep defense in depth.
		writeJSON(w, replyResponse{Text: ""})
		return
	}
	reply := generateReply(r.Context(), req)
	writeJSON(w, replyResponse{Text: reply})
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(payload)
}

func main() {
	flag.Parse()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"ok": true, "service": "personal-line-hakua-reply"})
	})
	mux.HandleFunc("/reply", handleReply)

	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("personal LINE Hakua reply server listening on http://%s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
