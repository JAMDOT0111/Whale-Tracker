package service

import (
	"context"
	"encoding/json"
	"eth-sweeper/model"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type AlertService struct {
	store     *AppStore
	etherscan *EtherscanClient
	notify    *NotifyService
}

func NewAlertService(store *AppStore, etherscan *EtherscanClient, notify *NotifyService) *AlertService {
	return &AlertService{store: store, etherscan: etherscan, notify: notify}
}

func (s *AlertService) NotificationStatus(ctx context.Context, userID string) map[string]any {
	return s.notify.Status(s.store.GetGmailToken(ctx, userID))
}

func (s *AlertService) SendTestNotification(ctx context.Context, userID string, email string) (model.NotificationLog, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		pref := s.store.GetPreference(ctx, userID)
		email = pref.Email
	}
	if email == "" {
		if user, ok := s.store.GetUser(ctx, userID); ok {
			email = user.Email
		}
	}
	now := nowISO()
	alert := model.AlertEvent{
		ID:           "test-" + stableID(userID+":"+email+":"+now),
		UserID:       userID,
		Address:      "0x00000000219ab540356cbb839cbe05303d7705fa",
		Type:         "test_notification",
		Severity:     "info",
		ThresholdETH: "500",
		Title:        "Test notification",
		Description:  "This is a real delivery test from ETH Whale Scanner.",
		Evidence: []model.Evidence{{
			TxHash:    "test-message",
			ValueETH:  "0",
			Asset:     "ETH",
			Timestamp: now,
			Reason:    "Manual notification test",
		}},
		Confidence: 1,
		Heuristic:  true,
		Status:     "open",
		DedupeKey:  "test:" + userID + ":" + now,
		CreatedAt:  now,
	}
	msgID, err := s.notify.SendAlert(ctx, email, alert, s.store.GetGmailToken(ctx, userID))
	if err != nil && strings.Contains(err.Error(), "gmail status 401") {
		if newToken, refreshedErr := s.refreshGmailToken(ctx, userID); refreshedErr == nil {
			msgID, err = s.notify.SendAlert(ctx, email, alert, newToken)
		} else {
			log.Printf("[alerts] failed to refresh token for user %s: %v", userID, refreshedErr)
		}
	}
	logEntry := model.NotificationLog{
		ID:       stableID(alert.ID + ":notification"),
		AlertID:  alert.ID,
		UserID:   userID,
		Channel:  "gmail",
		Status:   "sent",
		Attempts: 1,
	}
	if err != nil {
		logEntry.Status = "failed"
		logEntry.Error = err.Error()
		logEntry.NextRetryAt = nextRetryTime(1)
		s.store.AddNotificationLog(ctx, logEntry)
		return logEntry, err
	}
	logEntry.ProviderMessageID = msgID
	s.store.AddNotificationLog(ctx, logEntry)
	return logEntry, nil
}

func (s *AlertService) SendWatchlistConfirmation(ctx context.Context, userID string, item model.WatchlistItem) (model.NotificationLog, error) {
	pref := s.store.GetPreference(ctx, userID)
	email := pref.Email
	if email == "" {
		if user, ok := s.store.GetUser(ctx, userID); ok {
			email = user.Email
		}
	}
	now := nowISO()
	alert := model.AlertEvent{
		ID:           "watch-confirm-" + stableID(userID+":"+item.Address+":"+item.MinInteractionETH+":"+now),
		UserID:       userID,
		Address:      item.Address,
		Type:         "watchlist_confirmation",
		Severity:     "info",
		ThresholdETH: item.MinInteractionETH,
		Title:        "設定成功：地址異動監控",
		Description:  fmt.Sprintf("您已成功將 %s 加入追蹤名單，若交易金額大於 %s ETH 將會寄發通知。", labelOrAddress(item), item.MinInteractionETH),
		Evidence: []model.Evidence{{
			Asset:     "ETH",
			ValueETH:  item.MinInteractionETH,
			Timestamp: now,
			Reason:    "成功加入追蹤",
		}},
		Labels:     item.Labels,
		Confidence: 1,
		Heuristic:  true,
		Status:     "open",
		DedupeKey:  "watch-confirm:" + userID + ":" + item.Address + ":" + item.MinInteractionETH + ":" + now,
		CreatedAt:  now,
	}
	msgID, err := s.notify.SendAlert(ctx, email, alert, s.store.GetGmailToken(ctx, userID))
	if err != nil && strings.Contains(err.Error(), "gmail status 401") {
		if newToken, refreshedErr := s.refreshGmailToken(ctx, userID); refreshedErr == nil {
			msgID, err = s.notify.SendAlert(ctx, email, alert, newToken)
		} else {
			log.Printf("[alerts] failed to refresh token for user %s: %v", userID, refreshedErr)
		}
	}
	logEntry := model.NotificationLog{
		ID:       stableID(alert.ID + ":notification"),
		AlertID:  alert.ID,
		UserID:   userID,
		Channel:  "gmail",
		Status:   "sent",
		Attempts: 1,
	}
	if err != nil {
		logEntry.Status = "failed"
		logEntry.Error = err.Error()
		logEntry.NextRetryAt = nextRetryTime(1)
		s.store.AddNotificationLog(ctx, logEntry)
		return logEntry, err
	}
	logEntry.ProviderMessageID = msgID
	s.store.AddNotificationLog(ctx, logEntry)
	return logEntry, nil
}

func (s *AlertService) ScanWatchlists(ctx context.Context) int {
	items := s.store.AllWatchlists(ctx)
	created := 0
	for _, item := range items {
		if !item.NotificationOn {
			continue
		}
		threshold := parseFloatSafe(item.MinInteractionETH)
		if threshold <= 0 {
			threshold = 500
		}
		txs, err := s.etherscan.GetTransactions(item.Address, "", 50)
		if err != nil {
			log.Printf("[alerts] scan %s failed: %v", item.Address, err)
			continue
		}
		watchStartedAt := parseRFC3339OrZero(item.CreatedAt)
		for _, tx := range txs.Transactions {
			if strings.ToUpper(tx.Asset) != "ETH" {
				continue
			}
			if !isTransactionAfter(tx.Timestamp, watchStartedAt) {
				continue
			}
			value := parseFloatSafe(tx.Value)
			if value < threshold {
				continue
			}
			counterparty := tx.To
			direction := "out"
			if strings.EqualFold(tx.To, item.Address) {
				counterparty = tx.From
				direction = "in"
			}

			alert := model.AlertEvent{
				UserID:       item.UserID,
				Address:      item.Address,
				Type:         "watchlist_large_eth_transfer",
				Severity:     "info",
				ThresholdETH: item.MinInteractionETH,
				Title:        fmt.Sprintf("%s moved %s ETH", ShortAddress(item.Address), tx.Value),
				Description:  fmt.Sprintf("%s had a %sbound ETH transfer above the configured threshold.", labelOrAddress(item), direction),
				Evidence: []model.Evidence{{
					TxHash:       tx.Hash,
					From:         tx.From,
					To:           tx.To,
					ValueETH:     tx.Value,
					Asset:        tx.Asset,
					Timestamp:    tx.Timestamp,
					Counterparty: counterparty,
					Reason:       "watchlist threshold exceeded",
				}},
				Labels:     item.Labels,
				Confidence: 0.8,
				Heuristic:  true,
				DedupeKey:  item.UserID + ":" + item.Address + ":" + tx.Hash + ":" + item.MinInteractionETH,
			}
			alert, isNew := s.store.CreateAlert(ctx, alert)
			if !isNew {
				continue
			}
			created++
			s.sendAlert(ctx, alert)
		}
	}
	return created
}

func (s *AlertService) sendAlert(ctx context.Context, alert model.AlertEvent) {
	pref := s.store.GetPreference(ctx, alert.UserID)
	if !pref.GmailEnabled {
		return
	}
	email := pref.Email
	if email == "" {
		if user, ok := s.store.GetUser(ctx, alert.UserID); ok {
			email = user.Email
		}
	}
	msgID, err := s.notify.SendAlert(ctx, email, alert, s.store.GetGmailToken(ctx, alert.UserID))
	
	if err != nil && strings.Contains(err.Error(), "gmail status 401") {
		if newToken, refreshedErr := s.refreshGmailToken(ctx, alert.UserID); refreshedErr == nil {
			msgID, err = s.notify.SendAlert(ctx, email, alert, newToken)
		} else {
			log.Printf("[alerts] failed to refresh token for user %s: %v", alert.UserID, refreshedErr)
		}
	}

	logEntry := model.NotificationLog{
		AlertID:  alert.ID,
		UserID:   alert.UserID,
		Channel:  "gmail",
		Status:   "sent",
		Attempts: 1,
	}
	if err != nil {
		logEntry.Status = "failed"
		logEntry.Error = err.Error()
		logEntry.NextRetryAt = nextRetryTime(1)
	} else {
		logEntry.ProviderMessageID = msgID
	}
	s.store.AddNotificationLog(ctx, logEntry)
}

func (s *AlertService) StartScheduler(ctx context.Context) {
	if !strings.EqualFold(os.Getenv("ENABLE_JOBS"), "true") {
		log.Println("[jobs] scheduler disabled; set ENABLE_JOBS=true to run background scans")
		return
	}
	go func() {
		interval := configuredWatchlistScanInterval()
		log.Printf("[jobs] watchlist scanner enabled; interval=%s", interval)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				created := s.ScanWatchlists(ctx)
				log.Printf("[jobs] watchlist scan created %d alerts", created)
			}
		}
	}()
}

func (s *AlertService) refreshGmailToken(ctx context.Context, userID string) (model.GmailToken, error) {
	currentToken := s.store.GetGmailToken(ctx, userID)
	if currentToken.RefreshToken == "" {
		return currentToken, fmt.Errorf("no refresh token available")
	}

	clientID := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		return currentToken, fmt.Errorf("google oauth credentials missing")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("refresh_token", currentToken.RefreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return currentToken, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return currentToken, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return currentToken, fmt.Errorf("failed to refresh token: status %d, body: %s", resp.StatusCode, string(body))
	}

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return currentToken, err
	}

	expiry := ""
	if out.ExpiresIn > 0 {
		expiry = time.Now().UTC().Add(time.Duration(out.ExpiresIn) * time.Second).Format(time.RFC3339)
	}

	// Wait, I need a trick to update AppStore tokens without altering UpsertUser.
	// Oh, UpsertUser accepts GoogleLoginRequest and will update the token if AccessToken is present.
	if user, ok := s.store.GetUser(ctx, userID); ok {
		s.store.UpsertUser(ctx, model.GoogleLoginRequest{
			Email:             user.Email,
			Name:              user.Name,
			AvatarURL:         user.AvatarURL,
			GmailAccessToken:  out.AccessToken,
			GmailRefreshToken: currentToken.RefreshToken, // preserve the existing one
			GmailTokenExpiry:  expiry,
		})
	}
	return s.store.GetGmailToken(ctx, userID), nil
}

func configuredWatchlistScanInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("WATCHLIST_SCAN_INTERVAL"))
	if raw == "" {
		return time.Minute
	}
	interval, err := time.ParseDuration(raw)
	if err != nil || interval < time.Minute {
		log.Printf("[jobs] invalid WATCHLIST_SCAN_INTERVAL=%q; using 1m", raw)
		return time.Minute
	}
	return interval
}

func labelOrAddress(item model.WatchlistItem) string {
	if item.Alias != "" {
		return item.Alias
	}
	return ShortAddress(item.Address)
}

func parseRFC3339OrZero(raw string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}
	}
	return t
}

func isTransactionAfter(raw string, since time.Time) bool {
	if since.IsZero() {
		return true
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return t.After(since)
}
