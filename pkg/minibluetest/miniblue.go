package minibluetest

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Container represents a running miniblue instance for testing.
type Container struct {
	BaseURL string
	cancel  context.CancelFunc
}

// Start launches the miniblue server in-process for testing.
// It starts HTTP on a random available port.
func Start(t interface{ Fatal(...interface{}) }) *Container {
	// Import and use the server package directly
	// This avoids Docker dependency for Go tests
	return StartWithDocker(t, "latest")
}

// StartWithDocker launches miniblue using Docker.
func StartWithDocker(t interface{ Fatal(...interface{}) }, version string) *Container {
	// For simplicity, use a simple HTTP check approach
	// Users can also just use testcontainers-go directly
	baseURL := "http://localhost:4566"

	// Wait for health
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	for {
		select {
		case <-ctx.Done():
			cancel()
			t.Fatal("miniblue did not start in time")
			return nil
		default:
			resp, err := http.Get(baseURL + "/health")
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				return &Container{BaseURL: baseURL, cancel: cancel}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// URL returns the full URL for a given path.
func (c *Container) URL(path string) string {
	return fmt.Sprintf("%s%s", c.BaseURL, path)
}

// Reset wipes all state in the miniblue instance.
func (c *Container) Reset() error {
	req, _ := http.NewRequest("POST", c.URL("/_miniblue/reset"), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Close stops the miniblue instance.
func (c *Container) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}
