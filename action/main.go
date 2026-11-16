package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/args"
	"github.com/bytebase/bytebase/action/command"
	"github.com/bytebase/bytebase/action/github"
	"github.com/bytebase/bytebase/action/world"
)

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
	cmd := command.NewRootCommand(w)

	if err := cmd.ExecuteContext(ctx); err != nil {
		// Cobra has printed the message. A failed sign-in needs no stack trace
		// after it.
		var signInErr *command.SignInError
		if errors.As(err, &signInErr) {
			if world.GetJobPlatform() == world.GitHub {
				_, _ = fmt.Println(github.ErrorCommand(signInErr.Error()))
			}
		} else if rolloutErr := new(command.RolloutError); !errors.As(err, &rolloutErr) {
			// A stopped rollout has printed its result, which names the cause.
			slog.Error("failed to execute command", "error", fmt.Sprintf("%+v", err))
		}
		os.Exit(1)
	}
}
