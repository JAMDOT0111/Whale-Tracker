package service

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"eth-sweeper/model"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type NewsService struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cache      model.NewsResponse
}

const (
	gdeltETHNewsQuery      = "ethereum cryptocurrency"
	cointelegraphETHRSSURL = "https://cointelegraph.com/rss/tag/ethereum"
	liveNewsCacheTTL       = 30 * time.Minute
	unavailableNewsTTL     = 2 * time.Minute
)

func NewNewsService() *NewsService {
	return &NewsService{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *NewsService) GetETHNews(ctx context.Context) model.NewsResponse {
	s.mu.RLock()
	cached := s.cache
	s.mu.RUnlock()
	if cached.CachedAt != "" && time.Since(parseTimeOrZero(cached.CachedAt)) < newsCacheTTL(cached.Source) {
		return cached
	}

	resp, err := s.fetchGDELT(ctx)
	if err != nil || len(resp.Items) == 0 {
		resp, err = s.fetchCointelegraphETHRSS(ctx)
	}
	if err != nil || len(resp.Items) == 0 {
		resp = unavailableNews()
	}
	s.mu.Lock()
	s.cache = resp
	s.mu.Unlock()
	return resp
}

func newsCacheTTL(source string) time.Duration {
	if source == "news_sources_unavailable" {
		return unavailableNewsTTL
	}
	return liveNewsCacheTTL
}

func (s *NewsService) fetchGDELT(ctx context.Context) (model.NewsResponse, error) {
	q := url.Values{}
	q.Set("query", gdeltETHNewsQuery)
	q.Set("mode", "artlist")
	q.Set("format", "json")
	q.Set("maxrecords", "10")
	q.Set("sort", "hybridrel")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.gdeltproject.org/api/v2/doc/doc?"+q.Encode(), nil)
	if err != nil {
		return model.NewsResponse{}, err
	}
	req.Header.Set("User-Agent", "ETH-Whale-Scanner/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return model.NewsResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.NewsResponse{}, fmt.Errorf("gdelt status %d", resp.StatusCode)
	}

	var payload struct {
		Articles []struct {
			URL         string `json:"url"`
			Title       string `json:"title"`
			Seendate    string `json:"seendate"`
			SourceName  string `json:"sourceCountry"`
			Domain      string `json:"domain"`
			SocialImage string `json:"socialimage"`
		} `json:"articles"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return model.NewsResponse{}, err
	}

	items := make([]model.NewsItem, 0, len(payload.Articles))
	for _, article := range payload.Articles {
		if article.URL == "" || article.Title == "" {
			continue
		}
		source := article.Domain
		if source == "" {
			source = "GDELT"
		}
		items = append(items, model.NewsItem{
			ID:          stableID(article.URL),
			Title:       article.Title,
			URL:         article.URL,
			Source:      source,
			PublishedAt: gdeltTime(article.Seendate),
			Snippet:     "Public news link from GDELT. Full article content is not copied.",
		})
	}
	return model.NewsResponse{Items: items, Source: "gdelt_doc_api", CachedAt: nowISO()}, nil
}

func (s *NewsService) fetchCointelegraphETHRSS(ctx context.Context) (model.NewsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cointelegraphETHRSSURL, nil)
	if err != nil {
		return model.NewsResponse{}, err
	}
	req.Header.Set("User-Agent", "ETH-Whale-Scanner/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return model.NewsResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.NewsResponse{}, fmt.Errorf("cointelegraph rss status %d", resp.StatusCode)
	}

	var feed struct {
		Channel struct {
			Items []struct {
				Title   string `xml:"title"`
				Link    string `xml:"link"`
				PubDate string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return model.NewsResponse{}, err
	}

	items := make([]model.NewsItem, 0, len(feed.Channel.Items))
	for _, entry := range feed.Channel.Items {
		title := strings.TrimSpace(html.UnescapeString(entry.Title))
		link := strings.TrimSpace(entry.Link)
		if title == "" || link == "" {
			continue
		}
		items = append(items, model.NewsItem{
			ID:          stableID(link),
			Title:       title,
			URL:         link,
			Source:      "cointelegraph.com",
			PublishedAt: rssTime(entry.PubDate),
			Snippet:     "來自 Cointelegraph Ethereum RSS。請開啟原文查看完整報導。",
		})
		if len(items) >= 10 {
			break
		}
	}
	return model.NewsResponse{Items: items, Source: "cointelegraph_eth_rss", CachedAt: nowISO()}, nil
}

func unavailableNews() model.NewsResponse {
	return model.NewsResponse{
		Items:    []model.NewsItem{},
		Source:   "news_sources_unavailable",
		CachedAt: nowISO(),
	}
}

func gdeltTime(raw string) string {
	if raw == "" {
		return nowISO()
	}
	for _, layout := range []string{"20060102150405", "20060102T150405Z", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return raw
}

func rssTime(raw string) string {
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
