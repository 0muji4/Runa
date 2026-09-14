package itunes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/itunes"
	"github.com/google/go-cmp/cmp"
)

// fakeApple serves /lookup from *lookupBody and answers HEAD on the 600px artwork path per has600.
func fakeApple(t *testing.T, lookupBody *string, has600 bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/lookup", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("country"); got != "jp" {
			t.Errorf("lookup country = %q, want %q", got, "jp")
		}
		if got := r.URL.Query().Get("entity"); got != "song" {
			t.Errorf("lookup entity = %q, want %q", got, "song")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(*lookupBody))
	})
	mux.HandleFunc("/art/600x600bb.jpg", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("artwork check method = %s, want HEAD", r.Method)
		}
		if !has600 {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestClient_Lookup(t *testing.T) {
	t.Parallel()

	// BASE stands for the fake server's URL, so the artwork HEAD goes to it.
	const song = `{"kind":"song","trackId":1001,"trackName":"夜想曲","artistName":"月詠",
		"artworkUrl100":"BASE/art/100x100bb.jpg","previewUrl":"https://cdn/p.m4a","trackViewUrl":"https://music.apple.com/jp/x"}`
	track := func(artwork string) itunes.Track {
		return itunes.Track{
			TrackID: 1001, Title: "夜想曲", Artist: "月詠",
			ArtworkURL: artwork, PreviewURL: "https://cdn/p.m4a", StoreURL: "https://music.apple.com/jp/x",
		}
	}

	tests := []struct {
		name        string
		body        string
		has600      bool
		wantErr     bool
		wantArtwork string // BASE-relative; empty means no tracks
		wantBlank   bool   // preview/store dropped (non-https)
	}{
		{
			name:        "曲があり600pxのアートワークも取れる",
			body:        `{"resultCount":1,"results":[` + song + `]}`,
			has600:      true,
			wantArtwork: "BASE/art/600x600bb.jpg",
		},
		{
			name:        "600pxが無ければAppleが返した100pxのURLのまま",
			body:        `{"resultCount":1,"results":[` + song + `]}`,
			has600:      false,
			wantArtwork: "BASE/art/100x100bb.jpg",
		},
		{
			name:        "song以外の結果は捨てる",
			body:        `{"resultCount":2,"results":[{"wrapperType":"collection","collectionId":7},` + song + `]}`,
			has600:      true,
			wantArtwork: "BASE/art/600x600bb.jpg",
		},
		{
			name: "該当なしは空のスライス",
			body: `{"resultCount":0,"results":[]}`,
		},
		{
			name: "https以外のURLは捨てる",
			body: `{"resultCount":1,"results":[{"kind":"song","trackId":1001,"trackName":"夜想曲","artistName":"月詠",
				"artworkUrl100":"BASE/art/100x100bb.jpg","previewUrl":"http://cdn/p.m4a","trackViewUrl":"intent://x#Intent;end"}]}`,
			has600:      true,
			wantArtwork: "BASE/art/600x600bb.jpg",
			wantBlank:   true,
		},
		{
			name:    "JSONでない応答はエラー",
			body:    `<html>`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var body string
			srv := fakeApple(t, &body, tt.has600)
			body = strings.ReplaceAll(tt.body, "BASE", srv.URL)

			got, err := itunes.NewClient(srv.URL, srv.Client()).Lookup(t.Context(), []int64{1001})
			if tt.wantErr {
				if err == nil {
					t.Fatal("Lookup() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Lookup() error = %v, want nil", err)
			}
			want := []itunes.Track{}
			if tt.wantArtwork != "" {
				tr := track(strings.ReplaceAll(tt.wantArtwork, "BASE", srv.URL))
				if tt.wantBlank {
					tr.PreviewURL, tr.StoreURL = "", ""
				}
				want = append(want, tr)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Lookup() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClient_Lookup_EmptyIDsSkipsTheRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL)
	}))
	t.Cleanup(srv.Close)

	got, err := itunes.NewClient(srv.URL, srv.Client()).Lookup(t.Context(), nil)
	if err != nil || got != nil {
		t.Errorf("Lookup(nil) = (%v, %v), want (nil, nil)", got, err)
	}
}

func TestClient_Lookup_UpstreamFailureIsAnError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	if _, err := itunes.NewClient(srv.URL, srv.Client()).Lookup(t.Context(), []int64{1}); err == nil {
		t.Error("Lookup() error = nil, want an error for a 503")
	}
}

func TestLargerArtworkURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "100pxの区切りを600pxに置き換える",
			in:   "https://cdn/a/100x100bb.jpg",
			want: "https://cdn/a/600x600bb.jpg",
		},
		{
			name: "区切りが無ければそのまま",
			in:   "https://cdn/a/600x600bb.jpg",
			want: "https://cdn/a/600x600bb.jpg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := itunes.LargerArtworkURL(tt.in); got != tt.want {
				t.Errorf("LargerArtworkURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
