package docker

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func TestVersion(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "docker.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix socket: %v", err)
	}
	defer listener.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/version" {
				http.NotFound(writer, request)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"Version":"29.3.0","ApiVersion":"1.48","MinAPIVersion":"1.24"}`))
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Close()

	client := NewClient(socketPath)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	info, err := client.Version(ctx)
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if info.Version != "29.3.0" {
		t.Fatalf("info.Version = %q, want 29.3.0", info.Version)
	}
	if info.APIVersion != "1.48" {
		t.Fatalf("info.APIVersion = %q, want 1.48", info.APIVersion)
	}
}
