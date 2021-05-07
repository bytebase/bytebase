package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	return &Main{}
}

func (m *Main) Run() error {
	db := sqlite.NewDB(DSN)
	if err := db.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	m.db = db

	server := server.NewServer()
	server.AuthService = sqlite.NewAuthService(db)
	m.server = server
	if err := server.Run(); err != nil {
		return err
	}

	return nil
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	log.Println("Trying to stop bytebase...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if m.server != nil {
		log.Println("Trying to gracefully shutdown server...")
		m.server.Shutdown(ctx)
	}

	if m.db != nil {
		log.Println("Trying to close database connections...")
		if err := m.db.Close(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	m := NewMain()

	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if err := m.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cancel()
	}()

	// Execute program.
	if err := m.Run(); err != nil {
		if err != http.ErrServerClosed {
			m.Close()
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	log.Println("Bytebase stopped properly.")
}
