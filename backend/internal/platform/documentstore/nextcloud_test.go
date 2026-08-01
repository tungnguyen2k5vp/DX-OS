package documentstore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNextcloudPutUsesWebDAVAndAutoCreatesCollections(t *testing.T) {
	var receivedPath string
	var collectionCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "service user" || password != "secret" {
			t.Fatal("expected service account Basic authentication")
		}
		if r.Method == "MKCOL" {
			collectionCount++
			w.WriteHeader(http.StatusCreated)
			return
		}
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method %s", r.Method)
		}
		receivedPath = r.URL.EscapedPath()
		if r.Header.Get("X-NC-WebDAV-AutoMkcol") != "1" {
			t.Fatal("expected automatic WebDAV collection creation")
		}
		content, _ := io.ReadAll(r.Body)
		if string(content) != "document" {
			t.Fatalf("unexpected content %q", content)
		}
		w.Header().Set("ETag", `"etag-1"`)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	store := NewNextcloud(server.URL, "service user", "secret", "DX OS")
	etag, err := store.Put(
		context.Background(),
		"purchase-requests/request 1/file",
		"application/pdf",
		[]byte("document"),
		strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatalf("Put() unexpected error: %v", err)
	}
	if etag != "etag-1" {
		t.Fatalf("unexpected etag %q", etag)
	}
	want := "/remote.php/dav/files/service%20user/DX%20OS/purchase-requests/request%201/file"
	if receivedPath != want {
		t.Fatalf("unexpected WebDAV path %q, want %q", receivedPath, want)
	}
	if collectionCount != 3 {
		t.Fatalf("created %d collections, want 3", collectionCount)
	}
}

func TestNextcloudGetMapsNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	store := NewNextcloud(server.URL, "service", "secret", "DX-OS")
	_, err := store.Get(context.Background(), "purchase-requests/request/file")
	if err != ErrNotFound {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestNextcloudRejectsUnsafePath(t *testing.T) {
	store := NewNextcloud("http://nextcloud", "service", "secret", "DX-OS")
	if _, err := store.Get(context.Background(), "../secret"); err == nil {
		t.Fatal("Get() expected unsafe path error")
	}
}
