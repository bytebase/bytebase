package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytebase/bytebase/bin/server/cmd"
)

var (
	port    = 1234
	rootURL = fmt.Sprintf("http://localhost:%d/api", port)
)

type controller struct {
	main   *cmd.Main
	client *http.Client
	cookie string
}

// StartMain starts the main server.
func (ctl *controller) StartMain(ctx context.Context, dataDir string) error {
	// start main server.
	logger, err := cmd.GetLogger()
	if err != nil {
		return fmt.Errorf("failed to get logger, error %w", err)
	}
	defer logger.Sync()
	profile := cmd.GetTestProfile(dataDir)
	ctl.main = cmd.NewMain(profile, logger)

	errChan := make(chan error, 1)
	go func() {
		if err := ctl.main.Run(ctx); err != nil {
			errChan <- fmt.Errorf("failed to run main server, error %w", err)
		}
	}()

	if err := waitForServerStart(ctl.main, errChan); err != nil {
		return fmt.Errorf("failed to wait for server to start, error %w", err)
	}

	// initialize controller clients.
	ctl.client = &http.Client{}

	return nil
}

func waitForServerStart(m *cmd.Main, errChan <-chan error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.GetServer() == nil {
				continue
			}
			e := m.GetServer().GetEcho()
			if e == nil {
				continue
			}
			addr := e.ListenerAddr()
			if addr != nil && strings.Contains(addr.String(), ":") {
				return nil // was started
			}
		case err := <-errChan:
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		}
	}
}

func (ctl *controller) Close() error {
	if ctl.main != nil {
		return ctl.main.Close()
	}
	return nil
}

// Login will login as user demo@example.com and caches its cookie.
func (ctl *controller) Login() error {
	resp, err := ctl.client.Post(
		fmt.Sprintf("%s/auth/login/BYTEBASE", rootURL),
		"",
		strings.NewReader(`{"data":{"type":"loginInfo","attributes":{"email":"demo@example.com","password":"1024"}}}`))
	if err != nil {
		return fmt.Errorf("fail to post login request, error %w", err)
	}

	cookie := ""
	h := resp.Header.Get("Set-Cookie")
	parts := strings.Split(h, "; ")
	for _, p := range parts {
		if strings.HasPrefix(p, "access-token=") {
			cookie = p
			break
		}
	}
	if cookie == "" {
		return fmt.Errorf("unable to find access token in the login response headers")
	}
	ctl.cookie = cookie

	return nil
}
