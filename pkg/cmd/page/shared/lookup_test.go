package shared

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lroolle/atlas-cli/api"
)

func TestResolvePage_NumericID(t *testing.T) {
	ctx := context.Background()
	client := &api.ConfluenceClient{}

	tests := []struct {
		name     string
		ref      string
		spaceKey string
		want     string
		wantErr  bool
	}{
		{"numeric ID", "12345", "", "12345", false},
		{"large numeric ID", "9999999999", "", "9999999999", false},
		{"empty ref returns empty", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolvePage(ctx, client, tt.ref, tt.spaceKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolvePage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolvePage_TitleWithoutSpace(t *testing.T) {
	ctx := context.Background()
	client := &api.ConfluenceClient{}

	_, err := ResolvePage(ctx, client, "Some Page Title", "")
	if err == nil {
		t.Error("ResolvePage() expected error for title without space, got nil")
	}
	expected := "cannot resolve page by title without space"
	if err.Error()[:len(expected)] != expected {
		t.Errorf("ResolvePage() error = %v, want prefix %v", err, expected)
	}
}

func TestParseConfluenceURL_ViewPageAction(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": [{"id": "123", "title": "Test"}]}`))
	}))
	defer server.Close()

	client := api.NewConfluenceClient(server.URL, "user", "token")

	tests := []struct {
		name    string
		rawURL  string
		want    string
		wantErr bool
	}{
		{
			"viewpage.action with pageId",
			"https://wiki.example.com/pages/viewpage.action?pageId=12345",
			"12345",
			false,
		},
		{
			"viewpage.action with pageId and other params",
			"https://wiki.example.com/pages/viewpage.action?pageId=67890&spaceKey=TEST",
			"67890",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfluenceURL(ctx, client, tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfluenceURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseConfluenceURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConfluenceURL_InvalidURL(t *testing.T) {
	ctx := context.Background()
	client := &api.ConfluenceClient{}

	tests := []struct {
		name   string
		rawURL string
	}{
		{"no path pattern", "https://wiki.example.com/random/path"},
		{"empty path", "https://wiki.example.com/"},
		{"display without space/title", "https://wiki.example.com/display/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfluenceURL(ctx, client, tt.rawURL)
			if err == nil {
				t.Errorf("ParseConfluenceURL(%q) expected error, got nil", tt.rawURL)
			}
		})
	}
}
