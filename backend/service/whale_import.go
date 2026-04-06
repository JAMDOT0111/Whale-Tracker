package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"eth-sweeper/model"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ethAddressPattern = regexp.MustCompile(`(?i)0x[0-9a-f]{40}`)
var quickExportPattern = regexp.MustCompile(`(?s)const\s+quickExportAccountsData\s*=\s*'(.*?)';`)

const maxWhaleCSVBytes = 25 * 1024 * 1024

func (s *AppStore) ImportWhalesCSV(ctx context.Context, filename string, content []byte) (model.WhaleImportResponse, error) {
	rows, err := readCSVRows(content)
	if err != nil {
		return model.WhaleImportResponse{}, err
	}

	whales := make([]model.WhaleAccount, 0, len(rows))
	skipped := 0
	for rowIndex, row := range rows {
		if rowIndex == 0 && looksLikeHeader(row) {
			continue
		}

		whale, ok := parseWhaleRow(row, rowIndex+1)
		if !ok {
			skipped++
			continue
		}
		whales = append(whales, whale)
	}
	if len(whales) == 0 {
		return model.WhaleImportResponse{}, fmt.Errorf("no ethereum addresses found in CSV")
	}

	importedAt := nowISO()
	importID := stableID(filename + ":" + importedAt + ":" + strconv.Itoa(len(whales)))
	for i := range whales {
		whales[i].UpdatedAt = importedAt
		whales[i].Source = "etherscan_top_accounts_import"
		whales[i].Confidence = 0.95
		whales[i].EvidenceRef = importID
	}
	imported := s.UpsertWhales(ctx, whales)

	return model.WhaleImportResponse{
		ImportID:   importID,
		Imported:   imported,
		Skipped:    skipped,
		Source:     "etherscan_top_accounts_import",
		ImportedAt: importedAt,
	}, nil
}

func (s *AppStore) ImportWhalesFromURL(ctx context.Context, rawURL string) (model.WhaleImportResponse, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		rawURL = strings.TrimSpace(os.Getenv("ETHERSCAN_TOP_ACCOUNTS_CSV_URL"))
	}
	if rawURL == "" {
		return s.ImportWhalesFromEtherscanPages(ctx, configuredEtherscanPages())
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return model.WhaleImportResponse{}, fmt.Errorf("invalid CSV URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return model.WhaleImportResponse{}, err
	}
	req.Header.Set("User-Agent", "ETH-Whale-Scanner/1.0")
	req.Header.Set("Accept", "text/csv,text/plain,*/*")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return model.WhaleImportResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.WhaleImportResponse{}, fmt.Errorf("CSV URL returned status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxWhaleCSVBytes+1))
	if err != nil {
		return model.WhaleImportResponse{}, err
	}
	if len(content) > maxWhaleCSVBytes {
		return model.WhaleImportResponse{}, fmt.Errorf("CSV exceeds %d bytes", maxWhaleCSVBytes)
	}

	filename := path.Base(parsed.Path)
	if filename == "." || filename == "/" || filename == "" {
		filename = "etherscan-top-accounts.csv"
	}
	return s.ImportWhalesCSV(ctx, filename, content)
}

func (s *AppStore) ImportWhalesFromEtherscanPages(ctx context.Context, maxPages int) (model.WhaleImportResponse, error) {
	if maxPages <= 0 {
		maxPages = 20
	}
	if maxPages > 400 {
		maxPages = 400
	}

	client := &http.Client{Timeout: 30 * time.Second}
	all := make([]model.WhaleAccount, 0, maxPages*25)
	seen := map[string]bool{}
	for page := 1; page <= maxPages; page++ {
		pageWhales, err := fetchEtherscanAccountsPage(ctx, client, page)
		if err != nil {
			if page == 1 {
				return model.WhaleImportResponse{}, err
			}
			break
		}
		if len(pageWhales) == 0 {
			break
		}
		newRows := 0
		for _, whale := range pageWhales {
			if seen[whale.Address] {
				continue
			}
			seen[whale.Address] = true
			all = append(all, whale)
			newRows++
		}
		if newRows == 0 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if len(all) == 0 {
		return model.WhaleImportResponse{}, fmt.Errorf("no accounts parsed from Etherscan")
	}

	importedAt := nowISO()
	importID := stableID("etherscan-live:" + importedAt + ":" + strconv.Itoa(len(all)))
	for i := range all {
		all[i].UpdatedAt = importedAt
		all[i].Source = "etherscan_top_accounts_live"
		all[i].Confidence = 0.9
		all[i].EvidenceRef = importID
	}
	imported := s.UpsertWhales(ctx, all)
	return model.WhaleImportResponse{
		ImportID:   importID,
		Imported:   imported,
		Skipped:    0,
		Source:     "etherscan_top_accounts_live",
		ImportedAt: importedAt,
	}, nil
}

func fetchEtherscanAccountsPage(ctx context.Context, client *http.Client, page int) ([]model.WhaleAccount, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://etherscan.io/accounts/%d", page), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 ETH-Whale-Scanner/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("etherscan accounts page %d returned status %d", page, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxWhaleCSVBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxWhaleCSVBytes {
		return nil, fmt.Errorf("etherscan accounts page %d exceeds %d bytes", page, maxWhaleCSVBytes)
	}

	match := quickExportPattern.FindSubmatch(body)
	if len(match) < 2 {
		return nil, fmt.Errorf("quickExportAccountsData not found on Etherscan page %d", page)
	}
	rawJSON := decodeJSStringLiteral(string(match[1]))

	var rows []struct {
		Address    string `json:"Address"`
		NameTag    string `json:"NameTag"`
		Balance    string `json:"Balance"`
		Percentage string `json:"Percentage"`
		TxnCount   string `json:"TxnCount"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &rows); err != nil {
		return nil, fmt.Errorf("decode Etherscan accounts JSON page %d: %w", page, err)
	}

	whales := make([]model.WhaleAccount, 0, len(rows))
	for i, row := range rows {
		addr := strings.ToLower(strings.TrimSpace(row.Address))
		if !IsValidEthAddress(addr) {
			continue
		}
		balance := cleanETHBalance(row.Balance)
		if balance == "" {
			continue
		}
		txnCount, _ := parseIntLoose(row.TxnCount)
		whales = append(whales, model.WhaleAccount{
			Rank:       (page-1)*25 + i + 1,
			Address:    addr,
			NameTag:    strings.TrimSpace(row.NameTag),
			BalanceETH: balance,
			Percentage: strings.TrimSpace(row.Percentage),
			TxnCount:   txnCount,
		})
	}
	return whales, nil
}

func configuredEtherscanPages() int {
	raw := strings.TrimSpace(os.Getenv("ETHERSCAN_TOP_ACCOUNTS_PAGES"))
	if raw == "" {
		return 20
	}
	pages, err := strconv.Atoi(raw)
	if err != nil || pages <= 0 {
		return 20
	}
	return pages
}

func decodeJSStringLiteral(raw string) string {
	raw = strings.ReplaceAll(raw, `\'`, `'`)
	raw = strings.ReplaceAll(raw, `\"`, `"`)
	raw = strings.ReplaceAll(raw, `\\`, `\`)
	raw = strings.ReplaceAll(raw, `\n`, "\n")
	raw = strings.ReplaceAll(raw, `\r`, "\r")
	raw = strings.ReplaceAll(raw, `\t`, "\t")
	return raw
}

func readCSVRows(content []byte) ([][]string, error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty CSV")
	}

	reader := csv.NewReader(bytes.NewReader(trimmed))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err == nil {
		return rows, nil
	}

	// Etherscan exports can be copied as tabular text by hand. Support that as
	// a friendly fallback without adding a scraper.
	lines := strings.Split(string(trimmed), "\n")
	fallback := make([][]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fallback = append(fallback, regexp.MustCompile(`\s{2,}|\t`).Split(line, -1))
	}
	if len(fallback) == 0 {
		return nil, err
	}
	return fallback, nil
}

func looksLikeHeader(row []string) bool {
	joined := strings.ToLower(strings.Join(row, " "))
	return strings.Contains(joined, "address") || strings.Contains(joined, "balance") || strings.Contains(joined, "rank")
}

func parseWhaleRow(row []string, defaultRank int) (model.WhaleAccount, bool) {
	joined := strings.Join(row, " ")
	address := ethAddressPattern.FindString(joined)
	if !IsValidEthAddress(address) {
		return model.WhaleAccount{}, false
	}

	rank := defaultRank
	rankSet := false
	nameTag := ""
	balance := ""
	percentage := ""
	txnCount := 0

	for _, cell := range row {
		c := strings.TrimSpace(cell)
		if c == "" {
			continue
		}
		lower := strings.ToLower(c)
		if strings.EqualFold(c, address) || strings.Contains(lower, "address") {
			continue
		}
		if parsedRank, ok := parseIntLoose(c); ok && parsedRank > 0 && !rankSet {
			rank = parsedRank
			rankSet = true
			continue
		}
		if strings.Contains(lower, "eth") || (balance == "" && looksLikeAmount(c)) {
			if b := cleanETHBalance(c); b != "" {
				if balance == "" || parseFloatSafe(b) > parseFloatSafe(balance) {
					balance = b
				}
				continue
			}
		}
		if strings.Contains(c, "%") && percentage == "" {
			percentage = c
			continue
		}
		if n, ok := parseIntLoose(c); ok && n > 0 {
			txnCount = n
			continue
		}
		if nameTag == "" && !ethAddressPattern.MatchString(c) && !strings.Contains(strings.ToLower(c), "etherscan") {
			nameTag = c
		}
	}

	if balance == "" {
		return model.WhaleAccount{}, false
	}

	return model.WhaleAccount{
		Rank:       rank,
		Address:    strings.ToLower(address),
		NameTag:    nameTag,
		BalanceETH: balance,
		Percentage: percentage,
		TxnCount:   txnCount,
	}, true
}

func looksLikeAmount(cell string) bool {
	cleaned := strings.ReplaceAll(cell, ",", "")
	cleaned = strings.ReplaceAll(cleaned, "ETH", "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return false
	}
	_, err := strconv.ParseFloat(cleaned, 64)
	return err == nil
}

func cleanETHBalance(cell string) string {
	cleaned := strings.ToUpper(strings.TrimSpace(cell))
	cleaned = strings.ReplaceAll(cleaned, "ETH", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || value < 0 {
		return ""
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func parseIntLoose(cell string) (int, bool) {
	cleaned := strings.TrimSpace(cell)
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimPrefix(cleaned, "#")
	value, err := strconv.Atoi(cleaned)
	return value, err == nil
}
