package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"eth-sweeper/model"
	"eth-sweeper/service"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const googleOAuthStateCookie = "google_oauth_state"

type googleOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
}

type googleUserInfoResponse struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func (h *Handler) StartGoogleOAuth(c *gin.Context) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "GOOGLE_OAUTH_CLIENT_ID is not configured"})
		return
	}
	state := randomOAuthState()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(googleOAuthStateCookie, state, 600, "/", "", false, true)

	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("redirect_uri", googleRedirectURL(c))
	values.Set("response_type", "code")
	values.Set("scope", "openid email profile https://www.googleapis.com/auth/gmail.send")
	values.Set("access_type", "offline")
	values.Set("prompt", "consent")
	values.Set("include_granted_scopes", "true")
	values.Set("state", state)
	c.Redirect(http.StatusFound, "https://accounts.google.com/o/oauth2/v2/auth?"+values.Encode())
}

func (h *Handler) GoogleOAuthCallback(c *gin.Context) {
	if errMsg := c.Query("error"); errMsg != "" {
		c.Redirect(http.StatusFound, frontendAuthRedirect("auth_error", errMsg))
		return
	}
	stateCookie, err := c.Cookie(googleOAuthStateCookie)
	if err != nil || stateCookie == "" || stateCookie != c.Query("state") {
		c.Redirect(http.StatusFound, frontendAuthRedirect("auth_error", "invalid_oauth_state"))
		return
	}
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.Redirect(http.StatusFound, frontendAuthRedirect("auth_error", "missing_oauth_code"))
		return
	}

	tokenResp, err := exchangeGoogleOAuthCode(c.Request.Context(), code, googleRedirectURL(c))
	if err != nil {
		c.Redirect(http.StatusFound, frontendAuthRedirect("auth_error", "token_exchange_failed"))
		return
	}
	userInfo, err := fetchGoogleUserInfo(c.Request.Context(), tokenResp.AccessToken)
	if err != nil || userInfo.Email == "" {
		c.Redirect(http.StatusFound, frontendAuthRedirect("auth_error", "userinfo_failed"))
		return
	}

	expiry := ""
	if tokenResp.ExpiresIn > 0 {
		expiry = time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339)
	}
	user := h.store.UpsertUser(c.Request.Context(), model.GoogleLoginRequest{
		Email:             userInfo.Email,
		Name:              userInfo.Name,
		AvatarURL:         userInfo.Picture,
		GmailAccessToken:  tokenResp.AccessToken,
		GmailRefreshToken: tokenResp.RefreshToken,
		GmailTokenExpiry:  expiry,
	})
	h.store.UpsertPreference(c.Request.Context(), user.ID, model.NotificationPreference{
		UserID:       user.ID,
		Email:        user.Email,
		GmailEnabled: true,
		MinSeverity:  "info",
	})
	c.SetCookie(googleOAuthStateCookie, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, frontendAuthRedirect("auth_user_id", user.ID))
}

func (h *Handler) LoginGoogle(c *gin.Context) {
	var req model.GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	user := h.store.UpsertUser(c.Request.Context(), req)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) GetMe(c *gin.Context) {
	userID := userIDFromRequest(c)
	if user, ok := h.store.GetUser(c.Request.Context(), userID); ok {
		c.JSON(http.StatusOK, gin.H{"user": user, "notification_preferences": h.store.GetPreference(c.Request.Context(), userID)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": nil, "notification_preferences": h.store.GetPreference(c.Request.Context(), userID)})
}

func (h *Handler) ListWhales(c *gin.Context) {
	minBalance := parseMinBalance(c.Query("min_balance_eth"))
	sortKey := c.DefaultQuery("sort", "balance_desc")
	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := parsePositiveInt(c.DefaultQuery("page_size", "25"), 25)

	resp := h.store.ListWhales(c.Request.Context(), minBalance, sortKey, page, pageSize, userIDFromRequest(c))
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ImportEtherscanWhalesCSV(c *gin.Context) {
	var filename string
	var content []byte

	file, header, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		filename = header.Filename
		content, err = io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read uploaded CSV: " + err.Error()})
			return
		}
	} else {
		filename = c.GetHeader("X-Import-Filename")
		content, err = io.ReadAll(c.Request.Body)
		if err != nil || len(strings.TrimSpace(string(content))) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Upload a CSV file or send CSV text in the request body"})
			return
		}
	}
	if filename == "" {
		filename = "etherscan-top-accounts.csv"
	}

	resp, err := h.store.ImportWhalesCSV(c.Request.Context(), filename, content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ImportEtherscanWhalesURL(c *gin.Context) {
	var req model.WhaleImportURLRequest
	if c.Request.Body != nil && strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
			return
		}
	}

	resp, err := h.store.ImportWhalesFromURL(c.Request.Context(), req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetAddressDetail(c *gin.Context) {
	address := normalizeAddress(c.Param("address"))
	if !service.IsValidEthAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address"})
		return
	}

	var balance *model.BalanceResponse
	if resp, err := h.etherscan.GetBalance(address); err == nil {
		balance = resp
	}

	whale, ok := h.store.GetWhale(c.Request.Context(), address)
	var whalePtr *model.WhaleAccount
	if ok {
		whalePtr = &whale
	}
	labels := h.store.LabelsForAddress(c.Request.Context(), address)
	isTracked := false
	for _, item := range h.store.ListWatchlists(c.Request.Context(), userIDFromRequest(c)) {
		if strings.EqualFold(item.Address, address) {
			isTracked = true
			break
		}
	}

	c.JSON(http.StatusOK, model.AddressDetailResponse{
		Address:       address,
		Balance:       balance,
		Whale:         whalePtr,
		Labels:        labels,
		RiskScore:     heuristicRiskScore(labels),
		IsTracked:     isTracked,
		LastCheckedAt: handlerNow(),
	})
}

func (h *Handler) GetAddressTransactions(c *gin.Context) {
	address := normalizeAddress(c.Param("address"))
	if !service.IsValidEthAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address"})
		return
	}
	pageSize := parsePositiveInt(c.DefaultQuery("page_size", "100"), 100)
	pageKey := c.Query("page_key")
	resp, err := h.etherscan.GetTransactions(address, pageKey, pageSize)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error(), "transactions": []model.Transaction{}, "page_key": "", "total": 0})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetAddressGraph(c *gin.Context) {
	address := normalizeAddress(c.Param("address"))
	if !service.IsValidEthAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address"})
		return
	}
	resp, err := h.graph.BuildGraph(address)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error(), "nodes": []model.GraphNode{}, "edges": []model.GraphEdge{}})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetAddressAISummary(c *gin.Context) {
	address := normalizeAddress(c.Param("address"))
	if !service.IsValidEthAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address"})
		return
	}
	labels := h.store.LabelsForAddress(c.Request.Context(), address)
	summary := "No high-confidence market-moving claim is made. This address should be reviewed through its evidence transactions, labels, and balance changes."
	if whale, ok := h.store.GetWhale(c.Request.Context(), address); ok {
		summary = "This address is present in the imported Etherscan Top Accounts snapshot with " + whale.BalanceETH + " ETH. Treat any Smart Money or Market Mover label as heuristic and verify the listed transaction hashes."
	}
	c.JSON(http.StatusOK, model.AISummaryResponse{
		Address:    address,
		Summary:    summary,
		Heuristic:  true,
		Confidence: 0.55,
		Evidence: []model.Evidence{{
			Reason: "summary generated from imported whale snapshot and local labels only",
		}},
		CreatedAt: handlerNow(),
		Labels:    labels,
	})
}

func (h *Handler) GetETHPrices(c *gin.Context) {
	interval := c.DefaultQuery("interval", "5m")
	c.JSON(http.StatusOK, h.prices.GetETHSeries(c.Request.Context(), interval))
}

func (h *Handler) GetETHNews(c *gin.Context) {
	c.JSON(http.StatusOK, h.news.GetETHNews(c.Request.Context()))
}

func (h *Handler) GetCryptoFigureNews(c *gin.Context) {
	c.JSON(http.StatusOK, h.figures.GetCryptoFigureNews(c.Request.Context()))
}

func (h *Handler) ListWatchlists(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.store.ListWatchlists(c.Request.Context(), userIDFromRequest(c))})
}

func (h *Handler) UpsertWatchlist(c *gin.Context) {
	var req model.UpsertWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	item, err := h.store.UpsertWatchlist(c.Request.Context(), userIDFromRequest(c), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) UpsertWatchlistWithConfirmation(c *gin.Context) {
	var req model.UpsertWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	userID := userIDFromRequest(c)
	item, err := h.store.UpsertWatchlist(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logEntry, sendErr := h.alerts.SendWatchlistConfirmation(c.Request.Context(), userID, item)
	resp := gin.H{
		"item":                item,
		"notification_log":    logEntry,
		"notification_status": h.alerts.NotificationStatus(c.Request.Context(), userID),
	}
	if sendErr != nil {
		resp["notification_error"] = sendErr.Error()
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteWatchlist(c *gin.Context) {
	if ok := h.store.DeleteWatchlist(c.Request.Context(), userIDFromRequest(c), c.Param("id")); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Watchlist item not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListAlerts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.store.ListAlerts(c.Request.Context(), userIDFromRequest(c))})
}

func (h *Handler) MarkAlertRead(c *gin.Context) {
	if ok := h.store.MarkAlertRead(c.Request.Context(), userIDFromRequest(c), c.Param("id")); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) UpdateNotificationPreferences(c *gin.Context) {
	var pref model.NotificationPreference
	if err := c.ShouldBindJSON(&pref); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.store.UpsertPreference(c.Request.Context(), userIDFromRequest(c), pref))
}

func (h *Handler) GetNotificationStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.alerts.NotificationStatus(c.Request.Context(), userIDFromRequest(c)))
}

func (h *Handler) SendTestNotification(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if c.Request.Body != nil && strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
			return
		}
	}
	logEntry, err := h.alerts.SendTestNotification(c.Request.Context(), userIDFromRequest(c), req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":               err.Error(),
			"notification_status": h.alerts.NotificationStatus(c.Request.Context(), userIDFromRequest(c)),
			"log":                 logEntry,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":                  true,
		"log":                 logEntry,
		"notification_status": h.alerts.NotificationStatus(c.Request.Context(), userIDFromRequest(c)),
	})
}

func (h *Handler) RunWatchlistScan(c *gin.Context) {
	created := h.alerts.ScanWatchlists(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"alerts_created": created})
}

func userIDFromRequest(c *gin.Context) string {
	userID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if userID == "" {
		userID = strings.TrimSpace(c.Query("user_id"))
	}
	if userID == "" {
		userID = "anonymous"
	}
	return userID
}

func randomOAuthState() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func googleRedirectURL(c *gin.Context) string {
	if configured := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_REDIRECT_URL")); configured != "" {
		return configured
	}
	return "http://127.0.0.1:8080/api/auth/google/callback"
}

func frontendAuthRedirect(key string, value string) string {
	frontendURL := strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_URL")), "/")
	if frontendURL == "" {
		frontendURL = "http://127.0.0.1:5173"
	}
	values := url.Values{}
	values.Set(key, value)
	if key == "auth_user_id" {
		values.Set("auth", "google")
	}
	return frontendURL + "/?" + values.Encode()
}

func exchangeGoogleOAuthCode(ctx context.Context, code string, redirectURL string) (googleOAuthTokenResponse, error) {
	var out googleOAuthTokenResponse
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		return out, fmt.Errorf("google oauth client id/secret is not configured")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return out, fmt.Errorf("google token status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	if out.AccessToken == "" {
		return out, fmt.Errorf("missing google access token")
	}
	return out, nil
}

func fetchGoogleUserInfo(ctx context.Context, accessToken string) (googleUserInfoResponse, error) {
	var out googleUserInfoResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return out, fmt.Errorf("google userinfo status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

func normalizeAddress(address string) string {
	return strings.ToLower(strings.TrimSpace(address))
}

func parseMinBalance(raw string) float64 {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "TEH", "ETH")
	raw = strings.ReplaceAll(raw, "ETH", "")
	raw = strings.ReplaceAll(raw, ">", "")
	raw = strings.ReplaceAll(raw, ",", "")
	v, _ := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return v
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func heuristicRiskScore(labels []model.AddressLabelResult) model.RiskScore {
	score := 20
	level := "low"
	reasons := []string{"No high-confidence scam evidence in local data."}
	for _, label := range labels {
		switch label.Category {
		case "scam", "high_risk":
			score = 85
			level = "high"
			reasons = []string{"Address matched a high-risk label source: " + label.Source}
		case "unknown":
			if score < 35 {
				score = 35
				level = "medium"
				reasons = []string{"Address is not yet classified by local labels."}
			}
		case "exchange", "bridge", "defi_protocol":
			if score < 30 {
				score = 30
				level = "medium"
				reasons = []string{"Address interacts with or is classified as infrastructure: " + label.Category}
			}
		}
	}
	return model.RiskScore{
		Score:         score,
		Level:         level,
		Source:        "local_rules",
		Confidence:    0.45,
		Heuristic:     true,
		Reasons:       reasons,
		LastCheckedAt: handlerNow(),
	}
}

func handlerNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
