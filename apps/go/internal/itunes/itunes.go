// Package itunes fetches song metadata from Apple's iTunes Search API.
package itunes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// DefaultBaseURL is Apple's public lookup host.
const DefaultBaseURL = "https://itunes.apple.com"

// country pins lookups to the Japanese storefront; previews and store links differ per storefront.
const country = "jp"

// Apple returns only artworkUrl100; the 600px path rewrite is undocumented,
// so it is verified with a HEAD request and falls back to the 100px URL.
const (
	artworkSizeReturned = "100x100bb"
	artworkSizeWanted   = "600x600bb"
)

// artworkChecks bounds the concurrent artwork HEAD requests of one Lookup.
const artworkChecks = 8

// maxResponseBytes caps what Lookup reads from Apple (a 200-id lookup is well under 1 MiB).
const maxResponseBytes = 4 << 20

// Track is the metadata Runa keeps from a lookup. PreviewURL is empty when Apple
// offers no preview; every URL is https (anything else is dropped).
type Track struct {
	TrackID    int64
	Title      string
	Artist     string
	ArtworkURL string
	PreviewURL string
	StoreURL   string
}

// Client talks to the iTunes Search API.
type Client struct {
	client  *http.Client
	baseURL string
}

// NewClient builds a client for baseURL; client.Timeout bounds the lookup and each artwork HEAD.
func NewClient(baseURL string, client *http.Client) *Client {
	return &Client{
		client:  client,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// lookupResponse mirrors the fields Runa reads from GET /lookup.
type lookupResponse struct {
	Results []struct {
		Kind          string `json:"kind"`
		TrackID       int64  `json:"trackId"`
		TrackName     string `json:"trackName"`
		ArtistName    string `json:"artistName"`
		ArtworkURL100 string `json:"artworkUrl100"`
		PreviewURL    string `json:"previewUrl"`
		TrackViewURL  string `json:"trackViewUrl"`
	} `json:"results"`
}

// Lookup fetches the songs for ids in one request; unknown ids are simply absent from the result.
func (c *Client) Lookup(ctx context.Context, ids []int64) ([]Track, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	joined := make([]string, 0, len(ids))
	for _, id := range ids {
		joined = append(joined, strconv.FormatInt(id, 10))
	}
	q := url.Values{
		"id":      {strings.Join(joined, ",")},
		"country": {country},
		"entity":  {"song"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/lookup?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("itunes: build lookup request: %w", err)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("itunes: lookup: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("itunes: lookup: unexpected status %d", res.StatusCode)
	}

	var body lookupResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, maxResponseBytes)).Decode(&body); err != nil {
		return nil, fmt.Errorf("itunes: decode lookup response: %w", err)
	}

	tracks := make([]Track, 0, len(body.Results))
	for _, r := range body.Results {
		if r.Kind != "song" {
			continue
		}
		tracks = append(tracks, Track{
			TrackID:    r.TrackID,
			Title:      r.TrackName,
			Artist:     r.ArtistName,
			ArtworkURL: httpsOnly(r.ArtworkURL100),
			PreviewURL: httpsOnly(r.PreviewURL),
			StoreURL:   httpsOnly(r.TrackViewURL),
		})
	}
	c.upgradeArtwork(ctx, tracks)
	return tracks, nil
}

func httpsOnly(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	return raw
}

// upgradeArtwork swaps each track's artwork for the 600px rendition when the CDN confirms it.
func (c *Client) upgradeArtwork(ctx context.Context, tracks []Track) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, artworkChecks)
	for i := range tracks {
		wg.Add(1)
		sem <- struct{}{}
		go func(t *Track) {
			defer wg.Done()
			defer func() { <-sem }()
			t.ArtworkURL = c.largerArtwork(ctx, t.ArtworkURL)
		}(&tracks[i])
	}
	wg.Wait()
}

// LargerArtworkURL is the 600px rendition's URL for a 100px artwork URL (unchanged when
// there is no 100px segment). It does not check that the rendition exists; Lookup does.
func LargerArtworkURL(artworkURL string) string {
	return strings.Replace(artworkURL, artworkSizeReturned, artworkSizeWanted, 1)
}

// largerArtwork returns the 600px rendition when the CDN confirms it, else artworkURL.
func (c *Client) largerArtwork(ctx context.Context, artworkURL string) string {
	larger := LargerArtworkURL(artworkURL)
	if larger == artworkURL {
		return artworkURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, larger, nil)
	if err != nil {
		return artworkURL
	}
	res, err := c.client.Do(req)
	if err != nil {
		return artworkURL
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return artworkURL
	}
	return larger
}
