package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/sqlite"
)

type Main struct {
	Server *server.Server

	DB *sqlite.DB
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

	// r := gin.Default()
	// // Dont worry about this line just yet, it will make sense in the Dockerise bit!
	// r.Use(static.Serve("/", static.LocalFile("./web", true)))
	// api := r.Group("/api")
	// api.GET("/ping", func(c *gin.Context) {
	// 	c.JSON(200, gin.H{
	// 		"message": "pong from backend",
	// 	})
	// })

	// r.Run()
}

func NewMain() *Main {
	return &Main{
		Server: server.NewServer(),
		DB:     sqlite.NewDB(":memory:"),
	}
}

func (m *Main) Run() error {
	if err := m.DB.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	// m.Server.TodoService = sqlite.NewTodoService(m.DB)

	// if err := m.Server.Run(); err != nil {
	// 	return err
	// }

	return nil
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	// if m.Server != nil {
	// 	if err := m.Server.Close(); err != nil {
	// 		return err
	// 	}
	// }
	if m.DB != nil {
		if err := m.DB.Close(); err != nil {
			return err
		}
	}
	return nil
}
