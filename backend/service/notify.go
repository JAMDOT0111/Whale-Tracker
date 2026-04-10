package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"eth-sweeper/model"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type NotifyService struct {
	httpClient *http.Client
	token      string
	from       string
	dryRun     bool
	smtpHost   string
	smtpPort   string
	smtpUser   string
	smtpPass   string
}

func NewNotifyService() *NotifyService {
	token := strings.TrimSpace(os.Getenv("GMAIL_ACCESS_TOKEN"))
	smtpUser := strings.TrimSpace(os.Getenv("GMAIL_SMTP_USERNAME"))
	smtpPass := strings.TrimSpace(os.Getenv("GMAIL_SMTP_PASSWORD"))
	smtpHost := strings.TrimSpace(os.Getenv("GMAIL_SMTP_HOST"))
	smtpPort := strings.TrimSpace(os.Getenv("GMAIL_SMTP_PORT"))
	from := strings.TrimSpace(os.Getenv("GMAIL_FROM"))
	if smtpHost == "" && smtpUser != "" && smtpPass != "" {
		smtpHost = "smtp.gmail.com"
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if from == "" {
		from = smtpUser
	}
	return &NotifyService{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		token:      token,
		from:       from,
		dryRun:     strings.EqualFold(os.Getenv("GMAIL_DRY_RUN"), "true"),
		smtpHost:   smtpHost,
		smtpPort:   smtpPort,
		smtpUser:   smtpUser,
		smtpPass:   smtpPass,
	}
}

func (s *NotifyService) SendAlert(ctx context.Context, to string, alert model.AlertEvent, userToken ...model.GmailToken) (string, error) {
	if to == "" {
		return "", fmt.Errorf("missing notification email")
	}
	subject := "ETH 異動通知: " + alert.Title
	body := renderAlertEmail(alert)
	if s.dryRun {
		return "dry-run:" + alert.ID, nil
	}

	if len(userToken) > 0 && strings.TrimSpace(userToken[0].AccessToken) != "" {
		raw := buildMIMEMessage(to, to, subject, body)
		return s.sendGmailAPI(ctx, raw, alert.ID, strings.TrimSpace(userToken[0].AccessToken))
	}
	raw := buildMIMEMessage(s.from, to, subject, body)
	if s.smtpUser != "" && s.smtpPass != "" && s.smtpHost != "" {
		return s.sendSMTP(to, raw, alert.ID)
	}
	if s.token == "" {
		return "", fmt.Errorf("gmail is not configured: set GMAIL_ACCESS_TOKEN or GMAIL_SMTP_USERNAME/GMAIL_SMTP_PASSWORD")
	}

	return s.sendGmailAPI(ctx, raw, alert.ID, s.token)
}

func (s *NotifyService) Status(userToken ...model.GmailToken) map[string]any {
	provider := "not_configured"
	configured := false
	if s.dryRun {
		provider = "dry_run"
		configured = true
	} else if len(userToken) > 0 && strings.TrimSpace(userToken[0].AccessToken) != "" {
		provider = "google_oauth"
		configured = true
	} else if s.smtpUser != "" && s.smtpPass != "" && s.smtpHost != "" {
		provider = "smtp"
		configured = true
	} else if s.token != "" {
		provider = "gmail_api"
		configured = true
	}
	return map[string]any{
		"configured": configured,
		"provider":   provider,
		"dry_run":    s.dryRun,
		"from":       s.from,
	}
}

func (s *NotifyService) sendSMTP(to string, raw string, alertID string) (string, error) {
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPass, s.smtpHost)
	addr := s.smtpHost + ":" + s.smtpPort
	if err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(raw)); err != nil {
		return "", err
	}
	return "smtp:" + alertID, nil
}

func (s *NotifyService) sendGmailAPI(ctx context.Context, raw string, alertID string, token string) (string, error) {
	payload := map[string]string{
		"raw": base64.RawURLEncoding.EncodeToString([]byte(raw)),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://gmail.googleapis.com/gmail/v1/users/me/messages/send", bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if err != nil {
			return "", fmt.Errorf("gmail status %d", resp.StatusCode)
		}
		return "", fmt.Errorf("gmail status %d: %s", resp.StatusCode, compactGoogleError(string(body)))
	}

	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ID == "" {
		out.ID = "gmail:" + alertID
	}
	return out.ID, nil
}

func renderAlertEmail(alert model.AlertEvent) string {
	var b strings.Builder
	b.WriteString("資料來源: Etherscan API / 公開以太坊鏈上資料\n")
	b.WriteString(alert.Description)
	b.WriteString("\n\n")
	b.WriteString("地址: " + alert.Address + "\n")
	if strings.HasPrefix(strings.ToLower(alert.Address), "0x") {
		b.WriteString("地址連結: https://etherscan.io/address/" + alert.Address + "\n")
	}
	if alert.Type == "test_notification" {
		b.WriteString("監控門檻: N/A (發信系統連線測試)\n")
	} else {
		b.WriteString("監控門檻: > " + alert.ThresholdETH + " ETH\n")
	}
	b.WriteString("信心水準: " + fmt.Sprintf("%.0f%%", alert.Confidence*100) + "\n")
	b.WriteString("啟發性分析: 是。此為程式自動判斷，不構成投資建議，亦不代表確定的詐欺行為。\n\n")
	if len(alert.Evidence) > 0 {
		b.WriteString("關聯交易紀錄:\n")
		for _, ev := range alert.Evidence {
			b.WriteString("- 交易哈希 (Tx): " + ev.TxHash + "\n")
			if strings.HasPrefix(strings.ToLower(ev.TxHash), "0x") {
				b.WriteString("  連結: https://etherscan.io/tx/" + ev.TxHash + "\n")
			}
			b.WriteString("  金額: " + ev.ValueETH + " " + ev.Asset + "\n")
			if ev.Timestamp != "" {
				b.WriteString("  時間戳: " + ev.Timestamp + "\n")
			}
			if ev.From != "" {
				b.WriteString("  發送方 (From): " + ev.From + "\n")
			}
			if ev.To != "" {
				b.WriteString("  接收方 (To): " + ev.To + "\n")
			}
			if ev.Counterparty != "" {
				b.WriteString("  交易對手: " + ev.Counterparty + "\n")
			}
			b.WriteString("  原因: " + ev.Reason + "\n")
		}
	}
	b.WriteString("\n如需修改通知偏好，請至儀表板設定。\n")
	return b.String()
}

func buildMIMEMessage(from, to, subject, body string) string {
	if from == "" {
		from = "me"
	}
	subjectBase64 := base64.StdEncoding.EncodeToString([]byte(subject))
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: =?UTF-8?B?" + subjectBase64 + "?=",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}
	return strings.Join(headers, "\r\n")
}

func compactGoogleError(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "empty response body"
	}
	var parsed struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
			Details []struct {
				Reason string `json:"reason"`
				Domain string `json:"domain"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil && parsed.Error.Message != "" {
		msg := parsed.Error.Message
		if parsed.Error.Status != "" {
			msg = parsed.Error.Status + ": " + msg
		}
		if len(parsed.Error.Details) > 0 && parsed.Error.Details[0].Reason != "" {
			msg += " (" + parsed.Error.Details[0].Reason + ")"
		}
		return msg
	}
	if len(raw) > 700 {
		return raw[:700] + "..."
	}
	return raw
}

func nextRetryTime(attempts int) string {
	if attempts < 1 {
		attempts = 1
	}
	delay := time.Duration(attempts*attempts) * time.Minute
	return time.Now().UTC().Add(delay).Format(time.RFC3339)
}
