package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *HTTPClient) Execute(route *cli.Route, secretValue string) ([]byte, error) {
	switch {
	case route.Kind == cli.KindGateway && route.Action == "status":
		return c.get("/status")
	case route.Kind == cli.KindService && route.Action == "list":
		return c.get("/service/list")
	case route.Kind == cli.KindService && route.Action == "status":
		query := url.Values{}
		query.Set("service", route.Subject)
		return c.get("/service/status?" + query.Encode())
	case route.Kind == cli.KindService && route.Action == "deploy":
		return c.post("/service/deploy", cli.RouteRequest{Route: route, Service: route.Subject})
	case route.Kind == cli.KindService && route.Action == "restart":
		return c.post("/service/restart", cli.RouteRequest{Route: route, Service: route.Subject})
	case route.Kind == cli.KindService && route.Action == "remove":
		return c.post("/service/remove", cli.RouteRequest{Route: route, Service: route.Subject})
	case route.Kind == cli.KindService && route.Action == "logs":
		query := url.Values{}
		query.Set("service", route.Subject)
		return c.get("/service/logs?" + query.Encode())
	case route.Kind == cli.KindGateway && route.Action == "logs":
		return c.get("/logs")
	case route.Kind == cli.KindGateway && route.Action == "update":
		return c.post("/update", cli.RouteRequest{Route: route, Service: "gateway"})
	case route.Kind == cli.KindRuntimeNative:
		return c.post("/runtime/openclaw", cli.RouteRequest{Route: route})
	case route.Kind == cli.KindRuntimeVerify:
		return c.post("/runtime/verify", cli.RouteRequest{Route: route})
	case route.Kind == cli.KindServiceNative:
		return c.post("/service/passthrough", cli.RouteRequest{Route: route})
	default:
		return c.post("/execute", cli.RouteRequest{Route: route, SecretValue: secretValue})
	}
}

func (c *HTTPClient) get(path string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.do(request)
}

func (c *HTTPClient) post(path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	return c.do(request)
}

func (c *HTTPClient) delete(path string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.do(request)
}

func (c *HTTPClient) do(request *http.Request) ([]byte, error) {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", request.Method, request.URL.String(), err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read gateway response: %w", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("gateway returned an empty response for %s %s", request.Method, request.URL.Path)
	}

	return body, nil
}
