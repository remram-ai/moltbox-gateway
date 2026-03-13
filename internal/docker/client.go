package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Client struct {
	socketPath string
	httpClient *http.Client
}

type VersionInfo struct {
	Version       string `json:"Version"`
	APIVersion    string `json:"ApiVersion"`
	MinAPIVersion string `json:"MinAPIVersion"`
	GitCommit     string `json:"GitCommit"`
	GoVersion     string `json:"GoVersion"`
	OS            string `json:"Os"`
	Arch          string `json:"Arch"`
	KernelVersion string `json:"KernelVersion"`
}

func NewClient(socketPath string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Client) Version(ctx context.Context) (VersionInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/version", nil)
	if err != nil {
		return VersionInfo{}, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("query docker version over %s: %w", c.socketPath, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return VersionInfo{}, fmt.Errorf("docker version returned status %s", response.Status)
	}

	var info VersionInfo
	if err := json.NewDecoder(response.Body).Decode(&info); err != nil {
		return VersionInfo{}, fmt.Errorf("decode docker version response: %w", err)
	}

	return info, nil
}
