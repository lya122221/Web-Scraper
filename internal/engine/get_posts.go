package engine

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

var httpClient = newHTTPClient()

type RSSFeed struct {
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

func getPosts(ctx context.Context, url string) (*RSSFeed, error) {
	return getPostsWithClient(ctx, url, httpClient)
}

func getPostsWithClient(ctx context.Context, url string, client *http.Client) (*RSSFeed, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("bad response status code: %d", resp.StatusCode)
	}

	var rssFeed RSSFeed

	err = xml.NewDecoder(resp.Body).Decode(&rssFeed)
	if err != nil {
		return nil, err
	}

	return &rssFeed, nil
}

func newHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = safeDialContext

	return &http.Client{Transport: transport}
}

func safeDialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	for _, address := range addresses {
		if !address.IP.IsGlobalUnicast() || address.IP.IsPrivate() {
			return nil, fmt.Errorf("private network address is not allowed: %s", host)
		}
	}

	if len(addresses) == 0 {
		return nil, errors.New("host has no IP addresses")
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
}
