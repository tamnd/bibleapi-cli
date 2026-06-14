// Package bibleapi is the library behind the bibleapi command line:
// the HTTP client, request shaping, and the typed data models for bible-api.com.
//
// The Client here is the spine every command shares. It sets a real
// User-Agent, paces requests so a busy session stays polite, and retries the
// transient failures (429 and 5xx) that any public site throws under load.
package bibleapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Host is the site this client talks to.
const Host = "bible-api.com"

// BaseURL is the root every request is built from.
const BaseURL = "https://" + Host

// DefaultUserAgent identifies the client to bible-api.com.
const DefaultUserAgent = "bibleapi-cli/0.1 (tamnd87@gmail.com)"

// Client talks to bible-api.com over HTTP.
type Client struct {
	HTTP      *http.Client
	UserAgent string
	// Rate is the minimum gap between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int

	last time.Time
}

// NewClient returns a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 15 * time.Second},
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
	}
}

// Verse is the summary record for a single verse or passage lookup.
type Verse struct {
	Reference       string `json:"reference"        kit:"id"`
	Text            string `json:"text"`
	TranslationID   string `json:"translation_id"`
	TranslationName string `json:"translation_name"`
}

// VerseDetail is one individual verse within a passage.
type VerseDetail struct {
	BookID   string `json:"book_id"`
	BookName string `json:"book_name"`
	Chapter  int    `json:"chapter"`
	Verse    int    `json:"verse"`
	Text     string `json:"text"`
}

// apiResponse mirrors what bible-api.com returns.
type apiResponse struct {
	Reference       string        `json:"reference"`
	Verses          []VerseDetail `json:"verses"`
	Text            string        `json:"text"`
	TranslationID   string        `json:"translation_id"`
	TranslationName string        `json:"translation_name"`
	TranslationNote string        `json:"translation_note"`
}

// GetVerse fetches a verse or passage reference and returns a summary Verse.
// The reference should use spaces (e.g. "john 3:16" or "romans 8:28-30").
// translation is optional; pass "" to use the API default (web).
func (c *Client) GetVerse(ctx context.Context, reference, translation string) (*Verse, error) {
	resp, err := c.fetch(ctx, reference, translation)
	if err != nil {
		return nil, err
	}
	return &Verse{
		Reference:       resp.Reference,
		Text:            strings.TrimSpace(resp.Text),
		TranslationID:   resp.TranslationID,
		TranslationName: resp.TranslationName,
	}, nil
}

// GetPassage fetches a passage and returns each verse as a VerseDetail.
func (c *Client) GetPassage(ctx context.Context, reference, translation string) ([]VerseDetail, error) {
	resp, err := c.fetch(ctx, reference, translation)
	if err != nil {
		return nil, err
	}
	// Trim whitespace from each verse text.
	for i := range resp.Verses {
		resp.Verses[i].Text = strings.TrimSpace(resp.Verses[i].Text)
	}
	return resp.Verses, nil
}

// fetch makes the actual HTTP call and decodes the JSON response.
func (c *Client) fetch(ctx context.Context, reference, translation string) (*apiResponse, error) {
	url := buildURL(reference, translation)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &resp, nil
}

// buildURL constructs the bible-api.com request URL.
// Spaces in the reference are replaced with '+' as required by the API.
func buildURL(reference, translation string) string {
	ref := strings.ReplaceAll(strings.TrimSpace(reference), " ", "+")
	u := BaseURL + "/" + ref
	if translation != "" {
		u += "?translation=" + translation
	}
	return u
}

// Get fetches url and returns the response body. It paces and retries
// according to the client's settings.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
