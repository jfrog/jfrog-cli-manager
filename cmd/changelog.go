package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// Constants for configuration
const (
	DefaultPerPage = 30
	DefaultTimeout = 30 * time.Second
	UserAgent      = "jfvm/1.0"
	MaxConcurrent  = 5
)

// Shared HTTP client for better performance
var httpClient = &http.Client{Timeout: DefaultTimeout}

// GitHubRelease is a minimal struct for GitHub Releases API responses
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}

type noteResult struct {
	Tag  string
	Name string
	Body string
	Time time.Time
	Err  error
}

// returns up to 5 release notes between two tags (exclusive lower bound, inclusive upper),
// ordered by published_at ascending.
func FetchTopReleasesNotes(ctx context.Context, owner, repo, fromTag, toTag string) ([]noteResult, error) {
	// Input validation
	if owner == "" || repo == "" || fromTag == "" || toTag == "" {
		return nil, fmt.Errorf("owner, repo, fromTag, and toTag cannot be empty")
	}

	// Resolve boundary releases to get their published_at and canonical tag names
	fromRel, err := getReleaseByTag(ctx, owner, repo, fromTag)
	if err != nil {
		return nil, fmt.Errorf("error fetching fromTag %s: %w", fromTag, err)
	}
	toRel, err := getReleaseByTag(ctx, owner, repo, toTag)
	if err != nil {
		return nil, fmt.Errorf("error fetching toTag %s: %w", toTag, err)
	}

	// Determine lower/upper bounds by published_at (exclusive lower, inclusive upper)
	minDate := fromRel.PublishedAt
	maxDate := toRel.PublishedAt
	upper := toRel
	if minDate.After(maxDate) {
		minDate, maxDate = maxDate, minDate
		upper = fromRel
	}

	// Find last page of releases via Link header
	lastPage, err := getLastPageReleases(ctx, owner, repo, DefaultPerPage)
	if err != nil {
		return nil, err
	}
	if lastPage < 1 {
		lastPage = 1
	}

	// Binary search the page that contains maxDate between its first and last items
	startPage, err := findPageForDateReleases(ctx, owner, repo, DefaultPerPage, lastPage, maxDate)
	if err != nil {
		return nil, err
	}
	if startPage < 1 {
		startPage = 1
	}

	// Collect tag names up to 5 from startPage forward (older pages) while within (minDate, maxDate]
	tags := make([]string, 0, 5)
	// Ensure upper tag is included first
	tags = append(tags, upper.TagName)

	for page := startPage; len(tags) < 5 && page <= lastPage; page++ {
		pageReleases, err := listReleasesPage(ctx, owner, repo, page, DefaultPerPage)
		if err != nil {
			return nil, err
		}
		if len(pageReleases) == 0 {
			break
		}
		for _, r := range pageReleases {
			if r.TagName == "" {
				continue
			}
			// in window: published_at > minDate and <= maxDate
			if !r.PublishedAt.After(minDate) || r.PublishedAt.After(maxDate) {
				continue
			}
			// avoid adding the upper tag twice
			if r.TagName == upper.TagName {
				continue
			}
			tags = append(tags, r.TagName)
			if len(tags) >= 5 {
				break
			}
		}
		// Early stop if oldest on this page is <= minDate
		if pageReleases[len(pageReleases)-1].PublishedAt.Before(minDate) || pageReleases[len(pageReleases)-1].PublishedAt.Equal(minDate) {
			break
		}
	}

	if len(tags) == 0 {
		return nil, fmt.Errorf("no releases found in date range for %s/%s", owner, repo)
	}

	// Fetch release notes by tag concurrently for the collected tags
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(MaxConcurrent)
	results := make([]noteResult, len(tags))
	for i := range tags {
		i := i
		tag := tags[i]
		g.Go(func() error {
			rel, err := getReleaseByTag(gctx, owner, repo, tag)
			if err != nil {
				results[i] = noteResult{Tag: tag, Err: err}
				return nil // Continue with other fetches even if this one fails
			}
			results[i] = noteResult{Tag: rel.TagName, Name: rel.Name, Body: rel.Body, Time: rel.PublishedAt}
			return nil
		})
	}
	g.Wait() // Collect all results, ignore errors since we handle them individually

	// Filter successful results and sort by published time ascending
	var successful []noteResult
	for _, r := range results {
		if r.Err == nil {
			successful = append(successful, r)
		}
	}

	if len(successful) == 0 {
		return nil, fmt.Errorf("failed to fetch any release notes for tags in range")
	}

	sort.Slice(successful, func(i, j int) bool {
		return successful[i].Time.Before(successful[j].Time)
	})

	return successful, nil
}

// tag should have v appended in the string by the user
func getReleaseByTag(ctx context.Context, owner, repo, tag string) (GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, tag), nil)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("failed to fetch release by tag: %w", err)
	}
	defer resp.Body.Close()
	// Read response body once for all cases
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		// try with v-prefix if not already present
		if !strings.HasPrefix(tag, "v") {
			return getReleaseByTag(ctx, owner, repo, "v"+tag)
		}
		return GitHubRelease{}, fmt.Errorf("release not found for tag %s", tag)
	}
	if resp.StatusCode == http.StatusForbidden {
		if strings.Contains(string(body), "rate limit") {
			return GitHubRelease{}, fmt.Errorf("GitHub API rate limit exceeded. Please wait and try again, or authenticate your requests")
		}
		return GitHubRelease{}, fmt.Errorf("GitHub API access forbidden: %s", string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GitHubRelease{}, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}
	var rel GitHubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return GitHubRelease{}, fmt.Errorf("failed to parse release: %w", err)
	}
	return rel, nil
}

func listReleasesPage(ctx context.Context, owner, repo string, page, perPage int) ([]GitHubRelease, error) {
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d&page=%d", owner, repo, perPage, page), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error handling
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		if strings.Contains(string(body), "rate limit") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Please wait and try again, or authenticate your requests")
		}
		return nil, fmt.Errorf("GitHub API access forbidden: %s", string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var rels []GitHubRelease
	if err := json.Unmarshal(body, &rels); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}
	return rels, nil
}

func getLastPageReleases(ctx context.Context, owner, repo string, perPage int) (int, error) {
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d&page=1", owner, repo, perPage), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch first releases page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "rate limit") {
			return 0, fmt.Errorf("GitHub API rate limit exceeded. Please wait and try again, or authenticate your requests")
		}
		return 0, fmt.Errorf("GitHub API access forbidden: %s", string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}
	link := resp.Header.Get("Link")
	if link == "" {
		return 1, nil
	}
	last := parseLastPageFromLink(link)
	if last == 0 {
		return 1, nil
	}
	return last, nil
}

func findPageForDateReleases(ctx context.Context, owner, repo string, perPage, lastPage int, target time.Time) (int, error) {
	lo, hi := 1, lastPage
	for lo <= hi {
		mid := lo + (hi-lo)/2
		items, err := listReleasesPage(ctx, owner, repo, mid, perPage)
		if err != nil {
			return 0, err
		}
		if len(items) == 0 {
			return mid, nil
		}
		first := items[0].PublishedAt
		last := items[len(items)-1].PublishedAt
		// in-page if target <= first && target >= last
		if !target.After(first) && !target.Before(last) {
			return mid, nil
		}
		if target.After(first) {
			hi = mid - 1
			continue
		}
		// target.Before(last)
		lo = mid + 1
	}
	// closest page fallback
	if lo > lastPage {
		return lastPage, nil
	}
	if lo < 1 {
		return 1, nil
	}
	return lo, nil
}

func parseLastPageFromLink(link string) int {
	parts := strings.Split(link, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, "rel=\"last\"") {
			start := strings.Index(p, "<")
			end := strings.Index(p, ">")
			if start == -1 || end == -1 || end <= start+1 {
				continue
			}
			url := p[start+1 : end]
			qIdx := strings.LastIndex(url, "page=")
			if qIdx == -1 {
				continue
			}
			q := url[qIdx+5:]
			for i := 0; i < len(q); i++ {
				if q[i] < '0' || q[i] > '9' {
					q = q[:i]
					break
				}
			}
			n := 0
			for i := 0; i < len(q); i++ {
				c := q[i]
				if c < '0' || c > '9' {
					break
				}
				n = n*10 + int(c-'0')
			}
			return n
		}
	}
	return 1
}

// Removes "New Contributors" sections and download details to keep only core changes
func FilterReleaseNotes(body string) string {
	lines := strings.Split(body, "\n")
	var filteredLines []string

	skipNewContributors := false
	skipDetails := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Start skipping from "## New Contributors" section
		if strings.HasPrefix(trimmedLine, "## New Contributors") {
			skipNewContributors = true
			continue
		}

		// Stop skipping when we hit "**Full Changelog**" or another ## section
		if skipNewContributors && (strings.HasPrefix(trimmedLine, "**Full Changelog") ||
			(strings.HasPrefix(trimmedLine, "##") && !strings.HasPrefix(trimmedLine, "## New Contributors"))) {
			skipNewContributors = false
			// Include the Full Changelog line but skip other ## sections after New Contributors
			if strings.HasPrefix(trimmedLine, "**Full Changelog") {
				filteredLines = append(filteredLines, line)
			}
			continue
		}

		// Skip details section (downloads)
		if strings.HasPrefix(trimmedLine, "<details>") {
			skipDetails = true
			continue
		}

		if strings.HasPrefix(trimmedLine, "</details>") {
			skipDetails = false
			continue
		}

		// Skip lines if we're in a section to be filtered
		if skipNewContributors || skipDetails {
			continue
		}

		filteredLines = append(filteredLines, line)
	}

	return strings.Join(filteredLines, "\n")
}

// DisplayChangelogResults displays the changelog results with proper formatting and colors
func DisplayChangelogResults(releaseNotes []noteResult, version1, version2 string, fetchDuration time.Duration, noColor, showTiming bool) {
	// Setup colors using centralized color scheme (imported from compare.go)
	colors := NewColorScheme(noColor)

	// Display header
	fmt.Printf("╔══════════════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                           📖 CHANGELOG RESULTS                                        ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════════════╝\n\n")

	// Display timing information
	if showTiming {
		fmt.Printf("⏱️  FETCH TIMING: %v\n", fetchDuration)
		fmt.Printf("📊 RELEASES FOUND: %d release(s) between %s and %s\n\n",
			len(releaseNotes),
			colors.Blue.Sprint(version1),
			colors.Blue.Sprint(version2))
	}

	if len(releaseNotes) == 0 {
		fmt.Printf("ℹ️  No release notes found between versions %s and %s\n", version1, version2)
		return
	}

	// Display each release note
	for i, note := range releaseNotes {
		// Release header with divider
		fmt.Printf("╭─────────────────────────────────────────────────────────────────────────────────────╮\n")
		fmt.Printf("│ %s %s\n",
			colors.Green.Sprint("📦 RELEASE:"),
			colors.Blue.Sprintf("%s (%s)", note.Name, note.Tag))

		fmt.Printf("│ %s %s\n",
			colors.Cyan.Sprint("📅 PUBLISHED:"),
			colors.Yellow.Sprint(note.Time.Format("2006-01-02 15:04:05 MST")))

		fmt.Printf("╰─────────────────────────────────────────────────────────────────────────────────────╯\n")

		// Release body content
		if note.Body != "" {
			// Format the release notes (already filtered)
			lines := strings.Split(strings.TrimSpace(note.Body), "\n")
			fmt.Printf("\n")

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					fmt.Printf("\n")
					continue
				}

				// Add some styling for different types of content
				if strings.HasPrefix(line, "##") {
					// Section headers
					fmt.Printf("  %s\n", colors.Magenta.Sprint(line))
				} else if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
					// Bullet points
					fmt.Printf("    %s\n", colors.Cyan.Sprint(line))
				} else {
					// Regular content
					fmt.Printf("  %s\n", line)
				}
			}
		} else {
			fmt.Printf("\n  📝 No detailed release notes available for this version.\n")
		}

		// Add spacing between releases
		if i < len(releaseNotes)-1 {
			fmt.Printf("\n\n")
		}
	}

	// Summary footer
	fmt.Printf("\n\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════════════════════════════╗\n")

	fmt.Printf("║ ✅ Summary: Displaying only recent %d release(s) between %s → %s\n",
		len(releaseNotes),
		colors.Blue.Sprint(version1),
		colors.Blue.Sprint(version2))
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════════════╝\n")
}
