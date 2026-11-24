// Package translate converts text between languages through free public
// endpoints: Google's gtx endpoint first, MyMemory when Google is unreachable.
package translate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Result is one translated string plus what the provider detected.
type Result struct {
	Text     string `json:"text"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Provider string `json:"provider"`
}

// Client talks to the free translation endpoints.
type Client struct {
	http *http.Client
}

// NewClient returns a client with a short timeout: the panel is interactive,
// so a stalled request should fail fast and let the user retry.
func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 8 * time.Second}}
}

// Translate converts text into target (a code such as "zh-CN" or "en").
func (c *Client) Translate(ctx context.Context, text, target string) (*Result, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("translate: empty text")
	}
	// Google first: no key, good quality, and it reports the detected source.
	if res, err := c.google(ctx, text, target); err == nil {
		return res, nil
	}
	return c.mymemory(ctx, text, target)
}

const googleEndpoint = "https://translate.googleapis.com/translate_a/single"

func (c *Client) google(ctx context.Context, text, target string) (*Result, error) {
	query := url.Values{}
	query.Set("client", "gtx")
	query.Set("sl", "auto")
	query.Set("tl", target)
	query.Set("dt", "t")
	query.Set("q", text)
	body, err := c.fetch(ctx, googleEndpoint+"?"+query.Encode())
	if err != nil {
		return nil, err
	}
	return parseGoogle(body, target)
}

// parseGoogle reads the gtx payload:
// [[[translated, original, ...], ...], null, detectedSource, ...].
func parseGoogle(body []byte, target string) (*Result, error) {
	var payload []any
	if err := json.Unmarshal(body, &payload); err != nil || len(payload) < 3 {
		return nil, fmt.Errorf("translate: bad google response")
	}
	segments, _ := payload[0].([]any)
	var text strings.Builder
	for _, raw := range segments {
		segment, _ := raw.([]any)
		if len(segment) == 0 {
			continue
		}
		if part, ok := segment[0].(string); ok {
			text.WriteString(part)
		}
	}
	if text.Len() == 0 {
		return nil, fmt.Errorf("translate: empty google result")
	}
	source, _ := payload[2].(string)
	return &Result{Text: text.String(), Source: source, Target: target, Provider: "Google"}, nil
}

const myMemoryEndpoint = "https://api.mymemory.translated.net/get"

func (c *Client) mymemory(ctx context.Context, text, target string) (*Result, error) {
	query := url.Values{}
	query.Set("q", text)
	query.Set("langpair", "autodetect|"+target)
	body, err := c.fetch(ctx, myMemoryEndpoint+"?"+query.Encode())
	if err != nil {
		return nil, err
	}
	return parseMyMemory(body, target)
}

func parseMyMemory(body []byte, target string) (*Result, error) {
	var payload struct {
		ResponseData struct {
			Translated string `json:"translatedText"`
			Detected   string `json:"detectedLanguage"`
		} `json:"responseData"`
		Status int `json:"responseStatus"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Status != http.StatusOK {
		return nil, fmt.Errorf("translate: bad mymemory response")
	}
	if payload.ResponseData.Translated == "" {
		return nil, fmt.Errorf("translate: empty mymemory result")
	}
	return &Result{
		Text:     payload.ResponseData.Translated,
		Source:   payload.ResponseData.Detected,
		Target:   target,
		Provider: "MyMemory",
	}, nil
}

func (c *Client) fetch(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("translate: status %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 1<<20))
}
