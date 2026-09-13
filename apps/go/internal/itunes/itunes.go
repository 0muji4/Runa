// Package itunes fetches song metadata from Apple's iTunes Search API, the
// source of today's song (docs/dd/todays-song-itunes-preview.md, Q1). It is the
// only place that knows Apple's request/response shape, so a change on Apple's
// side is fixed here alone.
//
// Apple's Promo Content terms allow the preview and artwork only to promote the
// track, streamed and never cached, next to an Apple Music badge that links to
// StoreURL. This package returns the URLs; the clients honour the terms.
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

// DefaultBaseURL is Apple's public lookup host. Tests point BaseURL at a local
// server instead.
const DefaultBaseURL = "https://itunes.apple.com"

// country pins lookups to the Japanese catalog: Runa is Japan-only, and
// previews/store links differ per storefront.
const country = "jp"

// Artwork sizes. Apple's response carries at most artworkUrl100; the same CDN
// serves larger renditions when the size segment of the path is rewritten,
// which is common practice but undocumented — so the rewrite is verified with a
// HEAD request and falls back to the documented 100px URL (DD Q1).
const (
	artworkSizeReturned = "100x100bb"
	artworkSizeWanted   = "600x600bb"
)

// artworkChecks bounds the concurrent HEAD requests of one Lookup: an archive
// page can carry 50 tracks, and checking them one by one would take a round
// trip each.
const artworkChecks = 8

// maxResponseBytes caps what Lookup reads from Apple. A 200-id lookup is well
// under 1 MiB; anything larger is not a response Runa should try to parse.
const maxResponseBytes = 4 << 20

// Track is the metadata Runa keeps from a lookup. PreviewURL is empty when Apple
// offers no preview for the track; callers decide whether that is an error.
// Every URL is https — anything else in Apple's response is dropped, since the
// clients hand StoreURL to the OS to open and stream PreviewURL as-is.
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

// NewClient builds a client for baseURL (DefaultBaseURL in production) over
// client, whose Timeout bounds each request: the lookup and every artwork HEAD
// check. Tests pass an httptest TLS server's client.
func NewClient(baseURL string, client *http.Client) *Client {
	return &Client{
		client:  client,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// lookupResponse mirrors the fields Runa reads from GET /lookup. Results of other
// kinds (albums, artists) can appear for an id and are skipped.
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

// Lookup fetches the songs for ids in one request (Apple accepts a
// comma-separated list). Ids Apple does not know are simply absent from the
// result; a transport or non-2xx failure is returned as an error.
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

// httpsOnly returns raw when it parses as an absolute https URL, else "".
func httpsOnly(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	return raw
}

// upgradeArtwork swaps each track's artwork for the 600px rendition when the
// CDN confirms it exists (at most artworkChecks in flight). A check that fails
// or runs out of context leaves the URL Apple returned.
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

// LargerArtworkURL is the 600px rendition's URL for a 100px artwork URL, or
// artworkURL itself when it carries no 100px size segment. It does not check
// that the rendition exists; Lookup does, and a caller re-fetching a track can
// use it to tell "the check failed this time" from "Apple changed the artwork".
func LargerArtworkURL(artworkURL string) string {
	return strings.Replace(artworkURL, artworkSizeReturned, artworkSizeWanted, 1)
}

// largerArtwork returns the 600px rendition of artworkURL when the CDN confirms
// it exists, otherwise the URL Apple returned.
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
