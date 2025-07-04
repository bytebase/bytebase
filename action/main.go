package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bytebase/bytebase/action/args"
	"github.com/bytebase/bytebase/action/world"
	"github.com/bytebase/bytebase/backend/common/log"
)

func checkVersionCompatibility(client *Client, cliVersion string) {
	if cliVersion == "unknown" {
		slog.Warn("CLI version unknown, unable to check compatibility")
		return
	}

	actuatorInfo, err := client.getActuatorInfo()
	if err != nil {
		slog.Warn("Unable to get server version for compatibility check", "error", err)
		return
	}

	serverVersion := actuatorInfo.Version
	if serverVersion == "" {
		slog.Warn("Server version is empty, unable to check compatibility")
		return
	}

	if cliVersion == "latest" {
		slog.Warn("Using 'latest' CLI version. It is recommended to use a specific version like bytebase-action:" + serverVersion + " to match your Bytebase server version " + serverVersion)
		return
	}

	if cliVersion != serverVersion {
		slog.Warn("CLI version mismatch", "cliVersion", cliVersion, "serverVersion", serverVersion, "recommendation", "use bytebase-action:"+serverVersion+" to match your Bytebase server")
	} else {
		slog.Info("CLI version matches server version", "version", cliVersion)
	}
}

func main() {
	slog.Info("bytebase-action version " + args.Version + " built at commit " + args.Gitcommit)
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	// Trigger graceful shutdown on SIGINT or SIGTERM.
	// The default signal sent by the `kill` command is SIGTERM,
	// which is taken as the graceful shutdown signal for many systems, eg., Kubernetes, Gunicorn.
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		slog.Info(fmt.Sprintf("%s received.", sig.String()))
		cancel()
	}()

	w := world.NewWorld()
	cmd := NewRootCommand(w)

	if err := cmd.ExecuteContext(ctx); err != nil {
		slog.Error("failed to execute command", log.BBError(err))
		os.Exit(1)
	}
}
