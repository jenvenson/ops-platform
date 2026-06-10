// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package cmdb

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var aggregatedPackageBaseURL = func() string {
	if u := os.Getenv("AGGREGATED_PACKAGE_BASE_URL"); u != "" {
		return u
	}
	return "http://localhost:8888/aggregation/"
}()

var tarLinkPattern = regexp.MustCompile(`href=["']?([^"' >]+\.tar)["']?`)

type aggregatedPackageCandidate struct {
	URL          string
	LastModified time.Time
}

func findLatestAggregatedPackageURL() string {
	return findLatestAggregatedPackageURLFromBase(aggregatedPackageBaseURL)
}

func findLatestAggregatedPackageURLFromBase(baseURL string) string {
	links := listAggregatedPackageLinks(baseURL)
	if len(links) == 0 {
		return ""
	}

	candidates := make([]aggregatedPackageCandidate, 0, len(links))
	for _, link := range links {
		lastModified := fetchRemoteLastModified(link)
		candidates = append(candidates, aggregatedPackageCandidate{
			URL:          link,
			LastModified: lastModified,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		iHasTime := !candidates[i].LastModified.IsZero()
		jHasTime := !candidates[j].LastModified.IsZero()
		if iHasTime && jHasTime && !candidates[i].LastModified.Equal(candidates[j].LastModified) {
			return candidates[i].LastModified.After(candidates[j].LastModified)
		}
		if iHasTime != jHasTime {
			return iHasTime
		}
		return candidates[i].URL > candidates[j].URL
	})

	return candidates[0].URL
}

func listAggregatedPackageLinks(indexURL string) []string {
	req, err := http.NewRequest(http.MethodGet, indexURL, nil)
	if err != nil {
		log.Printf("Failed to create aggregation index request: %v", err)
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch aggregation index: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch aggregation index, status=%s", resp.Status)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read aggregation index response: %v", err)
		return nil
	}

	base, err := url.Parse(indexURL)
	if err != nil {
		log.Printf("Failed to parse aggregation base URL: %v", err)
		return nil
	}

	matches := tarLinkPattern.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	links := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		rawLink := strings.TrimSpace(match[1])
		if rawLink == "" {
			continue
		}

		parsedLink, err := url.Parse(rawLink)
		if err != nil {
			continue
		}

		resolvedLink := base.ResolveReference(parsedLink).String()
		if _, exists := seen[resolvedLink]; exists {
			continue
		}
		seen[resolvedLink] = struct{}{}
		links = append(links, resolvedLink)
	}

	return links
}

func fetchRemoteLastModified(fileURL string) time.Time {
	req, err := http.NewRequest(http.MethodHead, fileURL, nil)
	if err != nil {
		return time.Time{}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return time.Time{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}
	}

	lastModifiedHeader := resp.Header.Get("Last-Modified")
	if lastModifiedHeader == "" {
		return time.Time{}
	}

	lastModified, err := http.ParseTime(lastModifiedHeader)
	if err != nil {
		return time.Time{}
	}

	return lastModified
}