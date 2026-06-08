package security

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const webShortNucleiTimeout = 20 * time.Second

type DiscoveryOptions struct {
	MaxDepth     int
	MaxURLs      int
	SameHostOnly bool
	FetchScripts bool
	MaxScripts   int
}

type DiscoveredTarget struct {
	URL    string
	Kind   string
	Depth  int
	Source string
}

type discoveryCollector struct {
	baseURL *url.URL
	opts    DiscoveryOptions
	items   []DiscoveredTarget
	seen    map[string]struct{}
}

var (
	htmlHrefRe   = regexp.MustCompile(`(?i)(?:href|action)\s*=\s*["']([^"'#]+)["']`)
	htmlScriptRe = regexp.MustCompile(`(?i)<script[^>]+src\s*=\s*["']([^"']+)["']`)
	jsURLRe      = regexp.MustCompile("(?i)(https?://[^\"'`\\s<>]+|/[a-z0-9._~!$&()*+,;=:@%/\\-]+)")
	sitemapLocRe = regexp.MustCompile(`(?i)<loc>([^<]+)</loc>`)
)

func defaultDiscoveryOptions(config *WebScanConfig) DiscoveryOptions {
	opts := DiscoveryOptions{
		MaxDepth:     1,
		MaxURLs:      25,
		SameHostOnly: true,
		FetchScripts: true,
		MaxScripts:   4,
	}
	if config == nil {
		return opts
	}
	if config.DiscoveryMaxDepth > 0 {
		opts.MaxDepth = config.DiscoveryMaxDepth
	}
	if config.DiscoveryMaxURLs > 0 {
		opts.MaxURLs = config.DiscoveryMaxURLs
	}
	return opts
}

func verificationTargetLimit(config *WebScanConfig) int {
	if config == nil {
		return 8
	}
	if config.VerificationMaxTargets > 0 {
		return config.VerificationMaxTargets
	}
	return 8
}

func isDeepWebScan(config *WebScanConfig) bool {
	return config != nil && normalizeWebScanProfile(config.ScanProfile) == "deep"
}

func DiscoverWebTargets(entry string, session *WebSession, opts DiscoveryOptions) ([]DiscoveredTarget, error) {
	entry = sanitizeScanURL(entry)
	parsedEntry, err := url.Parse(entry)
	if err != nil {
		return nil, err
	}

	collector := &discoveryCollector{
		baseURL: parsedEntry,
		opts:    opts,
		items:   make([]DiscoveredTarget, 0, opts.MaxURLs),
		seen:    map[string]struct{}{},
	}
	collector.add(entry, "page", 0, "entry")

	body, contentType, err := fetchDiscoveryDocument(entry, session)
	if err == nil && looksLikeHTML(contentType, body) {
		scriptURLs := extractScriptURLs(body)
		for _, link := range extractHTMLTargets(body) {
			collector.add(link, classifyDiscoveredURL(link), 1, "html")
		}
		if opts.FetchScripts {
			limit := opts.MaxScripts
			if limit <= 0 {
				limit = 4
			}
			for i, scriptURL := range scriptURLs {
				if i >= limit {
					break
				}
				scriptBody, scriptType, fetchErr := fetchDiscoveryDocument(scriptURL, session)
				if fetchErr != nil || !looksLikeScript(scriptType, scriptURL) {
					continue
				}
				for _, link := range extractScriptTargets(scriptBody) {
					collector.add(link, classifyDiscoveredURL(link), 1, "script")
				}
			}
		}
	}

	for _, path := range []string{"/robots.txt", "/sitemap.xml"} {
		rootURL := collector.baseURL.ResolveReference(&url.URL{Path: path}).String()
		doc, _, fetchErr := fetchDiscoveryDocument(rootURL, session)
		if fetchErr != nil {
			continue
		}
		for _, link := range extractStructuredTargets(doc) {
			collector.add(link, classifyDiscoveredURL(link), 1, path)
		}
	}

	sort.SliceStable(collector.items, func(i, j int) bool {
		if collector.items[i].Depth != collector.items[j].Depth {
			return collector.items[i].Depth < collector.items[j].Depth
		}
		return collector.items[i].URL < collector.items[j].URL
	})
	return collector.items, nil
}

func (c *discoveryCollector) add(raw string, kind string, depth int, source string) {
	if c == nil || strings.TrimSpace(raw) == "" {
		return
	}
	if c.opts.MaxURLs > 0 && len(c.items) >= c.opts.MaxURLs {
		return
	}
	if depth > c.opts.MaxDepth {
		return
	}

	normalized, ok := normalizeDiscoveredURL(raw, c.baseURL, c.opts.SameHostOnly)
	if !ok {
		return
	}
	if isStaticDiscoveryAsset(normalized) {
		return
	}
	if _, exists := c.seen[normalized]; exists {
		return
	}

	c.seen[normalized] = struct{}{}
	c.items = append(c.items, DiscoveredTarget{
		URL:    normalized,
		Kind:   kind,
		Depth:  depth,
		Source: source,
	})
}

func fetchDiscoveryDocument(target string, session *WebSession) (string, string, error) {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "ops-platform-web-discovery/1.0")
	if session != nil {
		session.Apply(req)
	}

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("discovery request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", err
	}
	return string(body), resp.Header.Get("Content-Type"), nil
}

func normalizeDiscoveredURL(raw string, base *url.URL, sameHostOnly bool) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" ||
		strings.HasPrefix(raw, "#") ||
		strings.HasPrefix(strings.ToLower(raw), "javascript:") ||
		strings.HasPrefix(strings.ToLower(raw), "mailto:") ||
		strings.HasPrefix(strings.ToLower(raw), "tel:") {
		return "", false
	}

	ref, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	resolved := base.ResolveReference(ref)
	resolved.Fragment = ""

	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", false
	}
	if sameHostOnly && !sameDiscoveryHost(base, resolved) {
		return "", false
	}
	return sanitizeScanURL(resolved.String()), true
}

func sameDiscoveryHost(base *url.URL, target *url.URL) bool {
	if base == nil || target == nil {
		return false
	}
	return strings.EqualFold(base.Hostname(), target.Hostname())
}

func extractHTMLTargets(body string) []string {
	matches := htmlHrefRe.FindAllStringSubmatch(body, -1)
	results := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			results = append(results, match[1])
		}
	}
	return results
}

func extractScriptURLs(body string) []string {
	matches := htmlScriptRe.FindAllStringSubmatch(body, -1)
	results := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			results = append(results, match[1])
		}
	}
	return results
}

func extractScriptTargets(body string) []string {
	matches := jsURLRe.FindAllString(body, -1)
	results := make([]string, 0, len(matches))
	for _, match := range matches {
		candidate := strings.TrimSpace(match)
		if strings.HasPrefix(candidate, "/") &&
			!strings.Contains(candidate, "/api/") &&
			!strings.Contains(candidate, "/base/") &&
			!strings.Contains(candidate, "/ui/") {
			continue
		}
		results = append(results, candidate)
	}
	return results
}

func extractStructuredTargets(body string) []string {
	lines := strings.Split(body, "\n")
	results := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") || strings.HasPrefix(line, "/") {
			results = append(results, line)
		}
	}
	for _, match := range sitemapLocRe.FindAllStringSubmatch(body, -1) {
		if len(match) == 2 {
			results = append(results, strings.TrimSpace(match[1]))
		}
	}
	return results
}

func classifyDiscoveredURL(target string) string {
	lower := strings.ToLower(target)
	switch {
	case strings.Contains(lower, "/api/"), strings.Contains(lower, "/base/"):
		return "api"
	case strings.Contains(lower, "/auth/"):
		return "auth"
	default:
		return "page"
	}
}

func looksLikeHTML(contentType string, body string) bool {
	contentType = strings.ToLower(contentType)
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml+xml") {
		return true
	}
	snippet := strings.ToLower(strings.TrimSpace(body))
	return strings.Contains(snippet, "<html") || strings.Contains(snippet, "<body") || strings.Contains(snippet, "<script")
}

func looksLikeScript(contentType string, target string) bool {
	contentType = strings.ToLower(contentType)
	target = strings.ToLower(target)
	return strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "ecmascript") ||
		strings.HasSuffix(target, ".js")
}

func isStaticDiscoveryAsset(target string) bool {
	lower := strings.ToLower(target)
	pathLower := lower
	queryLower := ""
	if parsed, err := url.Parse(target); err == nil {
		pathLower = strings.ToLower(parsed.Path)
		queryLower = strings.ToLower(parsed.RawQuery)
	}

	staticSuffixes := []string{
		".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".woff", ".woff2", ".ttf", ".eot", ".map", ".txt",
	}
	for _, suffix := range staticSuffixes {
		if strings.HasSuffix(pathLower, suffix) {
			return true
		}
	}
	if isLowValueDiscoveryResource(pathLower, queryLower) {
		return true
	}
	return false
}

func isLowValueDiscoveryResource(pathLower string, queryLower string) bool {
	if pathLower == "" && queryLower == "" {
		return false
	}

	resourceKeywords := []string{
		"logo", "background", "favicon", "avatar", "watermark", "qrcode", "captcha",
	}
	for _, keyword := range resourceKeywords {
		if strings.Contains(pathLower, keyword) || strings.Contains(queryLower, keyword) {
			return true
		}
	}

	downloadHints := []string{"download", "preview", "export", "file-manage"}
	hasDownloadHint := false
	for _, hint := range downloadHints {
		if strings.Contains(pathLower, hint) {
			hasDownloadHint = true
			break
		}
	}
	if hasDownloadHint && (strings.Contains(queryLower, "uuid=") || strings.Contains(queryLower, "file_id=") || strings.Contains(queryLower, "fileid=")) {
		return true
	}
	if strings.Contains(queryLower, "uuid=default-") {
		return true
	}
	return false
}

func prioritizeVerificationTargets(config *WebScanConfig, items []DiscoveredTarget, limit int) ([]DiscoveredTarget, int) {
	if len(items) == 0 {
		return nil, 0
	}

	ranked := append([]DiscoveredTarget(nil), items...)
	sort.SliceStable(ranked, func(i, j int) bool {
		left := verificationTargetPriority(config, ranked[i])
		right := verificationTargetPriority(config, ranked[j])
		if left != right {
			return left > right
		}
		if ranked[i].Depth != ranked[j].Depth {
			return ranked[i].Depth < ranked[j].Depth
		}
		return ranked[i].URL < ranked[j].URL
	})

	if limit > 0 && len(ranked) > limit {
		return ranked[:limit], len(ranked) - limit
	}
	return ranked, 0
}

func verificationTargetPriority(config *WebScanConfig, item DiscoveredTarget) int {
	score := 0
	switch item.Kind {
	case "api":
		score += 300
	case "page":
		score += 200
	case "auth":
		score += 120
	default:
		score += 100
	}

	switch strings.ToLower(strings.TrimSpace(item.Source)) {
	case "entry":
		score += 1000
	case "browser-request":
		score += 220
	case "browser-dom":
		score += 140
	case "html":
		score += 120
	case "script":
		score += 110
	case "browser-frame":
		score += 80
	default:
		score += 40
	}

	if item.Depth == 0 {
		score += 80
	}
	if strings.Contains(strings.ToLower(item.URL), "/api/") || strings.Contains(strings.ToLower(item.URL), "/base/") {
		score += 60
	}
	if strings.EqualFold(strings.TrimSpace(item.Kind), "page") {
		score -= 90
	}
	if isRuleOnlyVerificationTarget(config, item) {
		score -= 160
	}
	return score
}

func shouldRunFullNucleiForTarget(config *WebScanConfig, item DiscoveredTarget) bool {
	return !isRuleOnlyVerificationTarget(config, item)
}

func isRuleOnlyVerificationTarget(config *WebScanConfig, item DiscoveredTarget) bool {
	if isDeepWebScan(config) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(item.Kind), "page") {
		return true
	}
	return isRuleOnlyAPIPath(item.URL)
}

func isRuleOnlyAPIPath(target string) bool {
	lower := strings.ToLower(strings.TrimSpace(target))
	if lower == "" {
		return false
	}

	pathLower := lower
	queryLower := ""
	if parsed, err := url.Parse(target); err == nil {
		pathLower = strings.ToLower(parsed.Path)
		queryLower = strings.ToLower(parsed.RawQuery)
	}

	for _, keyword := range []string{
		"/agreement", "agreement/",
		"/privacy", "privacy/",
		"/terms", "terms/",
		"/term", "term/",
		"get-term",
		"/policy", "policy/",
		"/license", "license/",
		"/open/plugin/installed/list",
		"/plugin/installed/list",
		"/user/verify/type/",
		"/verify/type/",
	} {
		if strings.Contains(pathLower, keyword) || strings.Contains(queryLower, keyword) {
			return true
		}
	}
	return false
}

func webVerificationNucleiTimeout(config *WebScanConfig, item DiscoveredTarget) time.Duration {
	if isDeepWebScan(config) {
		return nucleiCommandTimeout()
	}
	if isShortBudgetHighValueAPI(item.URL) {
		return webShortNucleiTimeout
	}
	return nucleiCommandTimeout()
}

func isShortBudgetHighValueAPI(target string) bool {
	lower := strings.ToLower(strings.TrimSpace(target))
	if lower == "" {
		return false
	}

	pathLower := lower
	if parsed, err := url.Parse(target); err == nil {
		pathLower = strings.ToLower(parsed.Path)
	}

	for _, keyword := range []string{
		"/base/common/get-func",
		"/base/custom/get",
	} {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	return false
}
