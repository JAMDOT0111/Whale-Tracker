package service

import (
	"context"
	"encoding/xml"
	"eth-sweeper/model"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	googleNewsRSSURL           = "https://news.google.com/rss/search"
	figureNewsLiveCacheTTL     = 30 * time.Minute
	figureNewsUnavailableTTL   = 2 * time.Minute
	figureNewsMaxItems         = 8
	figureNewsUnavailable      = "crypto_figure_news_unavailable"
	figureNewsNoMatches        = "crypto_figure_news_no_matches"
	figureNewsGoogleRSS        = "google_news_rss"
	figureNewsSnippetVitalik   = "來自 Google News RSS 的 Vitalik Buterin 加密貨幣相關新聞。請開啟原文查看完整報導。"
	figureNewsSnippetTrump     = "來自 Google News RSS 的 Trump 加密貨幣相關新聞。請開啟原文查看完整報導。"
	figureNewsDefaultUserAgent = "ETH-Whale-Scanner/1.0"
)

type figureNewsQuery struct {
	person  string
	query   string
	snippet string
	aliases []string
}

var cryptoFigureNewsQueries = []figureNewsQuery{
	{
		person:  "Vitalik Buterin",
		query:   "Vitalik Buterin Ethereum crypto cryptocurrency",
		snippet: figureNewsSnippetVitalik,
		aliases: []string{"vitalik", "buterin", "v神", "維塔利克", "布特林"},
	},
	{
		person:  "Donald Trump",
		query:   "Trump Bitcoin crypto cryptocurrency Ethereum",
		snippet: figureNewsSnippetTrump,
		aliases: []string{"trump", "donald trump", "川普", "特朗普"},
	},
}

var figureNewsCryptoKeywords = []string{
	"crypto",
	"cryptocurrency",
	"bitcoin",
	"ethereum",
	"btc",
	"eth",
	"加密貨幣",
	"加密",
	"比特幣",
	"以太坊",
	"虛擬貨幣",
}

type FigureNewsService struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cache      model.FigureNewsResponse
}

func NewFigureNewsService() *FigureNewsService {
	return &FigureNewsService{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *FigureNewsService) GetCryptoFigureNews(ctx context.Context) model.FigureNewsResponse {
	s.mu.RLock()
	cached := s.cache
	s.mu.RUnlock()
	if cached.CachedAt != "" && time.Since(parseTimeOrZero(cached.CachedAt)) < figureNewsCacheTTL(cached.Source) {
		return cached
	}

	resp, err := s.fetchCryptoFigureNews(ctx)
	if err != nil {
		resp = emptyFigureNewsResponse(figureNewsUnavailable)
	}

	s.mu.Lock()
	s.cache = resp
	s.mu.Unlock()
	return resp
}

func figureNewsCacheTTL(source string) time.Duration {
	if source == figureNewsGoogleRSS {
		return figureNewsLiveCacheTTL
	}
	return figureNewsUnavailableTTL
}

func (s *FigureNewsService) fetchCryptoFigureNews(ctx context.Context) (model.FigureNewsResponse, error) {
	items := make([]model.FigureNewsItem, 0, figureNewsMaxItems)
	successfulFeeds := 0
	seen := map[string]bool{}

	for _, query := range cryptoFigureNewsQueries {
		feedItems, err := s.fetchGoogleNewsRSS(ctx, query)
		if err != nil {
			continue
		}
		successfulFeeds++
		for _, item := range feedItems {
			key := strings.ToLower(strings.TrimSpace(item.URL))
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(item.Title))
			}
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			items = append(items, item)
		}
	}

	if successfulFeeds == 0 {
		return model.FigureNewsResponse{}, fmt.Errorf("all figure news feeds unavailable")
	}
	if len(items) == 0 {
		return emptyFigureNewsResponse(figureNewsNoMatches), nil
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].PublishedAt > items[j].PublishedAt
	})
	if len(items) > figureNewsMaxItems {
		items = items[:figureNewsMaxItems]
	}
	return model.FigureNewsResponse{
		Items:    items,
		Source:   figureNewsGoogleRSS,
		CachedAt: nowISO(),
	}, nil
}

func (s *FigureNewsService) fetchGoogleNewsRSS(ctx context.Context, figureQuery figureNewsQuery) ([]model.FigureNewsItem, error) {
	q := url.Values{}
	q.Set("q", figureQuery.query)
	q.Set("hl", "zh-TW")
	q.Set("gl", "TW")
	q.Set("ceid", "TW:zh-Hant")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleNewsRSSURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", figureNewsDefaultUserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google news rss status %d", resp.StatusCode)
	}

	var feed struct {
		Channel struct {
			Items []struct {
				Title   string `xml:"title"`
				Link    string `xml:"link"`
				PubDate string `xml:"pubDate"`
				Source  struct {
					Name string `xml:",chardata"`
				} `xml:"source"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	items := make([]model.FigureNewsItem, 0, len(feed.Channel.Items))
	for _, entry := range feed.Channel.Items {
		title := strings.TrimSpace(html.UnescapeString(entry.Title))
		link := strings.TrimSpace(entry.Link)
		if title == "" || link == "" {
			continue
		}
		if !isRelevantFigureNews(title, figureQuery) {
			continue
		}
		source := strings.TrimSpace(html.UnescapeString(entry.Source.Name))
		if source == "" {
			source = "Google News"
		}
		items = append(items, model.FigureNewsItem{
			ID:          stableID(figureQuery.person + "|" + link),
			Title:       title,
			URL:         link,
			Source:      source,
			PublishedAt: figureNewsRSSTime(entry.PubDate),
			Snippet:     figureQuery.snippet,
			Person:      figureQuery.person,
		})
	}
	return items, nil
}

func isRelevantFigureNews(title string, figureQuery figureNewsQuery) bool {
	normalized := strings.ToLower(title)
	return containsAny(normalized, figureQuery.aliases) && containsAny(normalized, figureNewsCryptoKeywords)
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func figureNewsRSSTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nowISO()
	}
	for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return raw
}

func emptyFigureNewsResponse(source string) model.FigureNewsResponse {
	return model.FigureNewsResponse{
		Items:    []model.FigureNewsItem{},
		Source:   source,
		CachedAt: nowISO(),
	}
}
