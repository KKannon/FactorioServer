package factorio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	factorioArchiveURL = "https://www.factorio.com/download/archive/"
	factorioLatestURL  = "https://factorio.com/api/latest-releases"
)

var (
	archiveVersionPattern = regexp.MustCompile(`/download/archive/([0-9]+\.[0-9]+\.[0-9]+)`)
	releaseHTTPClient     = &http.Client{Timeout: 12 * time.Second}
	releaseCacheMu        sync.RWMutex
	releaseCache          ReleaseCatalog
	releaseCacheTime      time.Time
)

// ReleaseCatalog is the list displayed by the server manager. Stable and
// Latest are aliases understood by the official Factorio download endpoint.
type ReleaseCatalog struct {
	Versions []string `json:"versions"`
	Stable   string   `json:"stable,omitempty"`
	Latest   string   `json:"latest,omitempty"`
}

type latestReleases struct {
	Stable       map[string]string `json:"stable"`
	Experimental map[string]string `json:"experimental"`
}

func parseArchiveVersions(reader io.Reader) ([]string, error) {
	body, err := io.ReadAll(io.LimitReader(reader, 2<<20))
	if err != nil {
		return nil, err
	}
	matches := archiveVersionPattern.FindAllSubmatch(body, -1)
	seen := make(map[string]bool, len(matches))
	versions := make([]string, 0, len(matches))
	for _, match := range matches {
		version := string(match[1])
		if !seen[version] {
			seen[version] = true
			versions = append(versions, version)
		}
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("Factorio archive did not contain any releases")
	}
	sort.Slice(versions, func(i, j int) bool { return compareReleaseVersions(versions[i], versions[j]) > 0 })
	return versions, nil
}

func compareReleaseVersions(left, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	for index := 0; index < 3; index++ {
		leftValue, _ := strconv.Atoi(leftParts[index])
		rightValue, _ := strconv.Atoi(rightParts[index])
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return 0
}

func fetchRelease(ctx context.Context, url string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json,text/html")
	request.Header.Set("User-Agent", "FactorioServerManager/0.2")
	response, err := releaseHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("Factorio release service returned HTTP %d", response.StatusCode)
	}
	return response, nil
}

func cacheReleaseCatalog(catalog ReleaseCatalog) {
	releaseCacheMu.Lock()
	releaseCache = catalog
	releaseCacheTime = time.Now()
	releaseCacheMu.Unlock()
}

// AvailableFactorioVersions obtains the exact releases from Factorio's
// official archive and adds the versions currently represented by its aliases.
func AvailableFactorioVersions(ctx context.Context) (ReleaseCatalog, error) {
	releaseCacheMu.RLock()
	cached := releaseCache
	cachedAt := releaseCacheTime
	releaseCacheMu.RUnlock()
	if len(cached.Versions) > 0 && time.Since(cachedAt) < 15*time.Minute {
		return cached, nil
	}

	archiveResponse, err := fetchRelease(ctx, factorioArchiveURL)
	if err != nil {
		if len(cached.Versions) > 0 {
			return cached, nil
		}
		return ReleaseCatalog{}, err
	}
	versions, err := parseArchiveVersions(archiveResponse.Body)
	archiveResponse.Body.Close()
	if err != nil {
		if len(cached.Versions) > 0 {
			return cached, nil
		}
		return ReleaseCatalog{}, err
	}

	catalog := ReleaseCatalog{Versions: versions}
	latestResponse, err := fetchRelease(ctx, factorioLatestURL)
	if err != nil {
		cacheReleaseCatalog(catalog)
		return catalog, nil
	}
	defer latestResponse.Body.Close()
	var releases latestReleases
	if err = json.NewDecoder(io.LimitReader(latestResponse.Body, 64<<10)).Decode(&releases); err != nil {
		cacheReleaseCatalog(catalog)
		return catalog, nil
	}
	catalog.Stable = releases.Stable["headless"]
	catalog.Latest = releases.Experimental["headless"]
	cacheReleaseCatalog(catalog)
	return catalog, nil
}
