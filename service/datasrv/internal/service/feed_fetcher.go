package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

type HTTPFeedFetcher struct {
	client *http.Client
}

func NewHTTPFeedFetcher(cfg conf.FeedSyncConfig) *HTTPFeedFetcher {
	return &HTTPFeedFetcher{
		client: &http.Client{Timeout: time.Duration(cfg.RequestTimeoutSeconds) * time.Second},
	}
}

func (f *HTTPFeedFetcher) Fetch(ctx context.Context, source dao.FeedSource, checkpoint dao.FeedCheckpoint) (FeedFetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return FeedFetchResult{}, fmt.Errorf("create request: %w", err)
	}
	if checkpoint.ETag != "" {
		req.Header.Set("If-None-Match", checkpoint.ETag)
	}
	if checkpoint.LastModified != "" {
		req.Header.Set("If-Modified-Since", checkpoint.LastModified)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return FeedFetchResult{}, fmt.Errorf("fetch feed: %w", err)
	}
	defer resp.Body.Close()

	result := FeedFetchResult{
		Source:       source,
		ETag:         resp.Header.Get("Etag"),
		LastModified: resp.Header.Get("Last-Modified"),
		FetchedAt:    time.Now().UTC(),
	}
	if resp.StatusCode == http.StatusNotModified {
		result.NotModified = true
		return result, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FeedFetchResult{}, fmt.Errorf("fetch feed: unexpected status %d", resp.StatusCode)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return FeedFetchResult{}, fmt.Errorf("read feed body: %w", err)
	}

	parsedSource, contents, err := parseFeedDocument(payload, source.ID)
	if err != nil {
		return FeedFetchResult{}, err
	}
	if parsedSource.DisplayName != "" {
		result.Source.DisplayName = parsedSource.DisplayName
	}
	if parsedSource.Description != "" {
		result.Source.Description = parsedSource.Description
	}
	if parsedSource.SiteURL != "" {
		result.Source.SiteURL = parsedSource.SiteURL
	}
	result.Contents = contents
	return result, nil
}

type feedEnvelope struct {
	XMLName xml.Name
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	GUID        string   `xml:"guid"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Content     string   `xml:"encoded"`
	Author      string   `xml:"author"`
	Categories  []string `xml:"category"`
	PubDate     string   `xml:"pubDate"`
	Updated     string   `xml:"date"`
}

type atomDocument struct {
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle"`
	Links    []atomLink  `xml:"link"`
	Entries  []atomEntry `xml:"entry"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type atomEntry struct {
	ID         string         `xml:"id"`
	Title      string         `xml:"title"`
	Summary    string         `xml:"summary"`
	Content    string         `xml:"content"`
	Updated    string         `xml:"updated"`
	Published  string         `xml:"published"`
	Links      []atomLink     `xml:"link"`
	Authors    []atomAuthor   `xml:"author"`
	Categories []atomCategory `xml:"category"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

func parseFeedDocument(payload []byte, sourceID string) (dao.FeedSource, []dao.FeedContent, error) {
	var env feedEnvelope
	if err := xml.Unmarshal(payload, &env); err != nil {
		return dao.FeedSource{}, nil, fmt.Errorf("parse feed envelope: %w", err)
	}

	switch env.XMLName.Local {
	case "rss":
		var doc rssDocument
		if err := xml.Unmarshal(payload, &doc); err != nil {
			return dao.FeedSource{}, nil, fmt.Errorf("parse rss feed: %w", err)
		}
		source := dao.FeedSource{
			ID:          sourceID,
			DisplayName: strings.TrimSpace(doc.Channel.Title),
			Description: strings.TrimSpace(doc.Channel.Description),
			SiteURL:     strings.TrimSpace(doc.Channel.Link),
		}
		contents := make([]dao.FeedContent, 0, len(doc.Channel.Items))
		for _, item := range doc.Channel.Items {
			publishedAt := parseFeedTime(item.PubDate, item.Updated)
			contents = append(contents, dao.FeedContent{
				FeedSourceID: sourceID,
				GUID:         strings.TrimSpace(item.GUID),
				Identity:     firstNonEmpty(strings.TrimSpace(item.GUID), strings.TrimSpace(item.Link), strings.TrimSpace(item.Title)+"|"+publishedAt.UTC().Format(time.RFC3339)),
				Title:        strings.TrimSpace(item.Title),
				Summary:      strings.TrimSpace(item.Description),
				Content:      firstNonEmpty(strings.TrimSpace(item.Content), strings.TrimSpace(item.Description)),
				Link:         strings.TrimSpace(item.Link),
				Author:       strings.TrimSpace(item.Author),
				Categories:   normalizeCategories(item.Categories),
				PublishedAt:  publishedAt,
				UpdatedAt:    publishedAt,
			})
		}
		return source, contents, nil
	case "feed":
		var doc atomDocument
		if err := xml.Unmarshal(payload, &doc); err != nil {
			return dao.FeedSource{}, nil, fmt.Errorf("parse atom feed: %w", err)
		}
		source := dao.FeedSource{
			ID:          sourceID,
			DisplayName: strings.TrimSpace(doc.Title),
			Description: strings.TrimSpace(doc.Subtitle),
			SiteURL:     pickAtomLink(doc.Links),
		}
		contents := make([]dao.FeedContent, 0, len(doc.Entries))
		for _, entry := range doc.Entries {
			publishedAt := parseFeedTime(entry.Published, entry.Updated)
			updatedAt := parseFeedTime(entry.Updated, entry.Published)
			contents = append(contents, dao.FeedContent{
				FeedSourceID: sourceID,
				GUID:         strings.TrimSpace(entry.ID),
				Identity:     firstNonEmpty(strings.TrimSpace(entry.ID), pickAtomLink(entry.Links), strings.TrimSpace(entry.Title)+"|"+publishedAt.UTC().Format(time.RFC3339)),
				Title:        strings.TrimSpace(entry.Title),
				Summary:      strings.TrimSpace(entry.Summary),
				Content:      firstNonEmpty(strings.TrimSpace(entry.Content), strings.TrimSpace(entry.Summary)),
				Link:         pickAtomLink(entry.Links),
				Author:       pickAtomAuthor(entry.Authors),
				Categories:   atomCategories(entry.Categories),
				PublishedAt:  publishedAt,
				UpdatedAt:    updatedAt,
			})
		}
		return source, contents, nil
	default:
		return dao.FeedSource{}, nil, fmt.Errorf("unsupported feed format %q", env.XMLName.Local)
	}
}

func parseFeedTime(values ...string) time.Time {
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC822Z,
		time.RFC822,
		time.RFC850,
		time.ANSIC,
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		for _, layout := range layouts {
			if parsed, err := time.Parse(layout, value); err == nil {
				return parsed.UTC()
			}
		}
	}
	return time.Time{}
}

func normalizeCategories(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func pickAtomLink(links []atomLink) string {
	for _, link := range links {
		if link.Rel == "" || link.Rel == "alternate" {
			return strings.TrimSpace(link.Href)
		}
	}
	if len(links) > 0 {
		return strings.TrimSpace(links[0].Href)
	}
	return ""
}

func pickAtomAuthor(authors []atomAuthor) string {
	for _, author := range authors {
		if strings.TrimSpace(author.Name) != "" {
			return strings.TrimSpace(author.Name)
		}
	}
	return ""
}

func atomCategories(categories []atomCategory) []string {
	out := make([]string, 0, len(categories))
	for _, category := range categories {
		if term := strings.TrimSpace(category.Term); term != "" {
			out = append(out, term)
		}
	}
	return out
}
