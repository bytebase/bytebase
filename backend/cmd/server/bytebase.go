package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/sqlite"
)

// const DSN = ":memory:"
const DSN = "./data/bytebase_dev.db"

type Main struct {
	server *server.Server

	db *sqlite.DB
}

func NewMain() *Main {
	return &Main{
		server: server.NewServer(),
		db:     sqlite.NewDB(DSN),
	}
}

func (m *Main) Run() error {
	if err := m.db.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	if err := m.server.Run(); err != nil {
		return err
	}

	return nil
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if m.server != nil {
		m.server.Close(ctx)
	}

	if m.db != nil {
		if err := m.db.Close(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := NewMain()

	// Execute program.
	if err := m.Run(); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	// Clean up program.
	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
