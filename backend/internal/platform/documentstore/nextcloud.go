package documentstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNotFound    = errors.New("document not found")
	ErrUnavailable = errors.New("document store unavailable")
)

type Nextcloud struct {
	baseURL  string
	username string
	password string
	root     string
	client   *http.Client
}

func NewNextcloud(baseURL, username, password, root string) *Nextcloud {
	return &Nextcloud{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		root:     root,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (n *Nextcloud) Put(
	ctx context.Context,
	path string,
	contentType string,
	content []byte,
	checksum string,
) (string, error) {
	if err := n.ensureCollections(ctx, path); err != nil {
		return "", err
	}
	request, err := n.request(ctx, http.MethodPut, path, bytes.NewReader(content))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("OC-Checksum", "SHA256:"+checksum)
	request.Header.Set("X-NC-WebDAV-AutoMkcol", "1")

	response, err := n.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: upload document: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf("%w: upload returned HTTP %d", ErrUnavailable, response.StatusCode)
	}
	return strings.Trim(response.Header.Get("ETag"), `"`), nil
}

func (n *Nextcloud) ensureCollections(ctx context.Context, path string) error {
	parts := strings.Split(path, "/")
	collections := []string{""}
	for index := 1; index < len(parts); index++ {
		collections = append(collections, strings.Join(parts[:index], "/"))
	}
	for _, collection := range collections {
		request, err := n.request(ctx, "MKCOL", collection, nil)
		if err != nil {
			return err
		}
		response, err := n.client.Do(request)
		if err != nil {
			return fmt.Errorf("%w: create document collection: %v", ErrUnavailable, err)
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		response.Body.Close()
		if response.StatusCode != http.StatusCreated &&
			response.StatusCode != http.StatusMethodNotAllowed {
			return fmt.Errorf(
				"%w: create collection returned HTTP %d",
				ErrUnavailable,
				response.StatusCode,
			)
		}
	}
	return nil
}

func (n *Nextcloud) Get(ctx context.Context, path string) ([]byte, error) {
	request, err := n.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	response, err := n.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: download document: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("%w: download returned HTTP %d", ErrUnavailable, response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 10*1024*1024+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read document: %v", ErrUnavailable, err)
	}
	if len(content) > 10*1024*1024 {
		return nil, fmt.Errorf("%w: stored document exceeds size limit", ErrUnavailable)
	}
	return content, nil
}

func (n *Nextcloud) Delete(ctx context.Context, path string) error {
	request, err := n.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	response, err := n.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: delete document: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("%w: delete returned HTTP %d", ErrUnavailable, response.StatusCode)
	}
	return nil
}

func (n *Nextcloud) request(
	ctx context.Context,
	method string,
	path string,
	body io.Reader,
) (*http.Request, error) {
	segments := []string{"remote.php", "dav", "files", n.username, n.root}
	if path != "" {
		for _, segment := range strings.Split(path, "/") {
			if segment == "" || segment == "." || segment == ".." {
				return nil, errors.New("document path contains an unsafe segment")
			}
			segments = append(segments, segment)
		}
	}
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		n.baseURL+"/"+strings.Join(segments, "/"),
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("create document request: %w", err)
	}
	request.SetBasicAuth(n.username, n.password)
	return request, nil
}
