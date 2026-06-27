package htmltomarkdown

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/yudaprama/tools/htmltomarkdown/converter"
	"github.com/yudaprama/tools/htmltomarkdown/internal/domutils"
	"github.com/yudaprama/tools/htmltomarkdown/internal/textutils"
	"github.com/yudaprama/tools/htmltomarkdown/plugin/base"
	"github.com/yudaprama/tools/htmltomarkdown/plugin/commonmark"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// ConvertString converts a html-string to a structured result.
func ConvertString(htmlInput string, opts ...converter.ConvertOptionFunc) (*CrawlSuccessResult, error) {
	doc, err := html.Parse(strings.NewReader(htmlInput))
	if err != nil {
		return nil, err
	}
	return ConvertNode(doc, opts...)
}

// ConvertReader converts the html from the reader to a structured result.
func ConvertReader(r io.Reader, opts ...converter.ConvertOptionFunc) (*CrawlSuccessResult, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	return ConvertNode(doc, opts...)
}

// ConvertURL fetches a URL and converts it to a structured result.
func ConvertURL(urlStr string, client *http.Client, opts ...converter.ConvertOptionFunc) (*CrawlSuccessResult, error) {
	if client == nil {
		client = http.DefaultClient
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	// Browser headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "br") // Only brotli - gzip/deflate auto-handled by http.Client
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Handle content encoding
	var bodyReader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "br" {
		bodyReader = brotli.NewReader(resp.Body)
	}

	utf8Reader, err := charset.NewReader(bodyReader, resp.Header.Get("Content-Type"))
	if err != nil {
		utf8Reader = bodyReader
	}

	// Limit to 5MB
	limitedReader := io.LimitReader(utf8Reader, 5*1024*1024)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if !textutils.IsValidUTF8Content(body) {
		return nil, fmt.Errorf("response contains invalid/binary content")
	}

	// Default usage: infer domain from URL if not provided
	if converter.GetOptions(opts...).Domain == "" {
		opts = append(opts, converter.WithDomain(extractWebsite(urlStr)))
	}
	opts = append(opts, converter.WithURL(urlStr))

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	// Try normal conversion
	if converter.GetOptions(opts...).MainContentOnly {
		domutils.RemoveNavigation(doc)
	}
	result, err := ConvertNode(doc, opts...)
	if err != nil {
		// Fallback to text extraction
		title := domutils.GetTitle(doc)
		website := extractWebsite(converter.GetOptions(opts...).Domain)

		result = &CrawlSuccessResult{
			Title:   title,
			Content: domutils.GetTextContent(doc),
			URL:     urlStr,
			Website: website,
			Crawler: converter.GetOptions(opts...).Crawler,
		}
	}

	result.Content = textutils.CleanMarkdownContent(result.Content)

	if len(result.Content) < 50 {
		return nil, fmt.Errorf("content too short (%d chars)", len(result.Content))
	}

	return result, nil
}

// ConvertNode converts a `*html.Node` to a structured result.
func ConvertNode(doc *html.Node, opts ...converter.ConvertOptionFunc) (*CrawlSuccessResult, error) {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
		),
	)

	markdown, err := conv.ConvertNode(doc, opts...)
	if err != nil {
		return nil, err
	}

	// Extract options to fill the result
	option := converter.GetOptions(opts...)
	title := domutils.GetTitle(doc)
	website := extractWebsite(option.Domain)

	return &CrawlSuccessResult{
		Title:   title,
		Content: string(markdown),
		URL:     option.URL,
		Website: website,
		Crawler: option.Crawler,
	}, nil
}

// extractWebsite extracts the website hostname from URL
func extractWebsite(domain string) string {
	if domain == "" {
		return ""
	}

	// Use the exported ParseBaseDomain from converter package
	u := converter.ParseBaseDomain(domain)
	if u == nil {
		if strings.Contains(domain, "/") {
			return strings.Split(domain, "/")[0]
		}
		return domain
	}

	return u.Hostname()
}
