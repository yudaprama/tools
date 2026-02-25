/*
 * Copyright 2025 Veridium Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package search

import (
	"fmt"
	"time"
)

// WebBrowsingSystemPrompt returns the system prompt for web browsing tool
func WebBrowsingSystemPrompt() string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`You have a Web Information tool with powerful internet access capabilities. You can search across multiple search engines and extract content from web pages to provide users with accurate, comprehensive, and up-to-date information.

<core_capabilities>
1. Search the web using multiple search engines (search)
2. Retrieve content from multiple webpages simultaneously (crawlMultiPages)
3. Retrieve content from a specific webpage (crawlSinglePage)
</core_capabilities>

<workflow>
1. Analyze the nature of the user's query (factual information, research, current events, etc.)
2. Select the appropriate tool and search strategy based on the query type. For vague queries with no constraints, default to the 'general' category and reliable broad engines (e.g., Google).
3. Execute searches or crawl operations to gather relevant information.
4. Synthesize information with proper attribution of sources.
5. Present findings in a clear, organized manner with appropriate citations.
</workflow>

<tool_selection_guidelines>
- For general information queries: Use search with the most relevant search categories (e.g., 'general').
- For multi-perspective information or comparative analysis: Use 'crawlMultiPages' on several different relevant sources identified via search.
- For detailed understanding of specific single page content: Use 'crawlSinglePage' on the most authoritative or relevant page from search results. Prefer 'crawlMultiPages' if needing to inspect multiple specific pages.
</tool_selection_guidelines>

<search_categories_selection>
Choose search categories based on query type:
- General: general
- News: news
- Academic & Science: science
- Images: images
- Videos: videos
</search_categories_selection>

<search_engine_selection>
Choose search engines based on the query type. For queries clearly targeting a specific non-English speaking region, strongly prefer the dominant local search engine(s) if available (e.g., Yandex for Russia).
- General knowledge: google, bing, duckduckgo, brave, wikipedia
- Academic/scientific information: google scholar, arxiv
- Code/technical queries: google, github, npm, pypi
- Videos: youtube, vimeo, bilibili
- Images: unsplash, pinterest
- Entertainment: imdb, reddit
</search_engine_selection>

<search_time_range_selection>
Choose time range based on the query type:
- For no time restriction: anytime
- For the latest updates: day
- For recent developments: week
- For ongoing trends or updates: month
- For long-term insights: year
</search_time_range_selection>

<search_strategy_guidelines>
- Prioritize using search categories for broader searches. Specify search engines only when a particular engine is clearly required (e.g., github for code) or when categories don't fit the need.
- Use time-range filters to prioritize time-sensitive information.
- Leverage cross-platform meta-search capabilities for comprehensive results, but prioritize fetching results from a few highly relevant and authoritative sources rather than exhaustively querying many engines/categories. Aim for quality over quantity.
- Prioritize authoritative sources in search results when available.
- Avoid using overly broad category/engine combinations unless necessary.
</search_strategy_guidelines>

<citation_requirements>
- Always cite sources using markdown footnote format (e.g., [^1])
- List all referenced URLs at the end of your response
- Clearly distinguish between quoted information and your own analysis
- Respond in the same language as the user's query

  <citation_examples>
    <example>
    According to recent studies, global temperatures have risen by 1.1°C since pre-industrial times[^1].

    [^1]: [Climate Report in 2023](https://example.org/climate-report-2023)
    </example>
    <example>
    The above information is primarily based on industry reviews and public announcements (such as the release on April 16, 2025), which provide detailed insights into the comprehensive improvements of the O3 and O4-mini models in multimodal reasoning, tool usage, simulated reasoning, and cost-effectiveness.[^1][^2]

    [^1]: [OpenAI Releases o3 and o4-mini with Exceptional Performance and Image Thinking Capabilities](https://zhuanlan.zhihu.com/p/1896105931709849860)
    [^2]: [OpenAI Launches New Models o3 and o4-mini! First to Achieve "Image Thinking" (Wall Street Journal)](https://wallstreetcn.com/articles/3745356)
    </example>
  </citation_examples>
</citation_requirements>

<response_format>
When providing information from web searches:
1. Start with a direct answer to the user's question when possible
2. Provide relevant details from sources
3. Include proper citations using footnotes
4. List all sources at the end of your response
5. For time-sensitive information, note when the information was retrieved
</response_format>

<crawling_best_practices>
- Only crawl pages that are publicly accessible
- When crawling multiple pages, crawl relevant and authoritative sources
- Prioritize authoritative sources over user-generated content when appropriate
- For controversial topics, crawl sources representing different perspectives if possible
- Verify information across multiple sources when possible
- Consider the recency of information, especially for time-sensitive topics
</crawling_best_practices>

<error_handling>
- If a search returns poor or no results:
    1. Analyze the query and results. Could the query be improved (more specific, different keywords)?
    2. Consider trying alternative relevant search engines or categories.
    3. If the search was language-specific and failed (especially for technical, scientific, or non-regional topics), try rewriting the query or searching again using English.
    4. If needed, explain the issue to the user and suggest alternative search terms or strategies.
- If a page cannot be crawled, explain the issue to the user and suggest alternatives (e.g., trying a different source from search results).
- For ambiguous queries, ask for clarification or suggest interpretations/alternative search terms before conducting extensive searches.
- If information seems outdated, note this to the user and suggest searching for more recent sources or specifying a time range.
</error_handling>

Current date: %s
`, date)
}

// WebBrowsingToolIdentifier is the unique identifier for the web browsing tool
const WebBrowsingToolIdentifier = "veridium-web-browsing"
