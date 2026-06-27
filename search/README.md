# Search Service

Web search dengan automatic fallback: Brave → DuckDuckGo.

## Problem & Solution

**Problem**: Brave API rate limit (429) menyebabkan "No results found"  
**Solution**: DuckDuckGo sebagai fallback otomatis

## How It Works

```
User Query → Brave API → Success? Return results
                       ↓ Fail (429)
              DuckDuckGo API → Success? Return results
                            ↓ Fail
                       Return error
```

## Providers

| Provider | API Key | Rate Limit | Quality | Use Case |
|----------|---------|------------|---------|----------|
| Brave | Required | Yes (strict) | ⭐⭐⭐⭐⭐ | Primary |
| DuckDuckGo | No | No | ⭐⭐⭐ | Fallback |

## Usage

```go
service := search.NewService()
response, err := service.WebSearch(search.SearchQuery{
    Query: "latest AI news",
})
```

Tidak perlu konfigurasi tambahan - fallback otomatis!

## Logs

```
🔍 Search provider: Brave (query=..., results=15)
⚠️  Brave search failed: status 429. Falling back to DuckDuckGo...
🔍 Search provider: DuckDuckGo (query=..., results=12)
❌ All search providers failed for query: ...
```

## Testing

```bash
go test ./internal/search -v
```

## Crawler Title Extraction

Crawler menggunakan fallback hierarchy untuk extract title:

1. `<title>` tag
2. `<meta property="og:title">`
3. `<meta name="twitter:title">`
4. First `<h1>` tag
5. `<meta name="description">` (truncated)
6. **Generate from URL** (jika semua gagal)

**URL to Title Examples:**
- `https://techcrunch.com/2025/12/19/openai-news/` → `Openai News`
- `https://www.nature.com/articles/d41586-025-03936-2` → `D41586 025 03936 2`
- `https://example.com` → `example.com`

## Troubleshooting

**"No results found" masih muncul?**
- Check logs untuk error message
- Verify network connectivity
- Jika ada TLS error (internetpositif.id), gunakan VPN

**Title masih URL?**
- Seharusnya sudah otomatis di-convert ke human-readable
- Check log: `⚠️ [crawlNaive] Empty title for URL, generating from URL`
