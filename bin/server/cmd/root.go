package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/store"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// -----------------------------------Command Line Config BEGIN------------------------------------

var (
	// Used for flags.
	mode    string
	host    string
	dataDir string
	port    int

	rootCmd = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase server",
		Run: func(cmd *cobra.Command, args []string) {
			if mode != "release" && mode != "dev" && mode != "demo" {
				fmt.Fprintln(os.Stderr, fmt.Errorf("invalid --mode %s, expect \"release\" | \"dev\" | \"demo\"", mode))
				os.Exit(1)
			}
			// Trim trailing / in case user supplies
			dir, err := filepath.Abs(strings.TrimRight(dataDir, "/"))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			dataDir = dir
			start()
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&mode, "mode", "release", "mode which Bytebase is running under. release | dev | demo")
	rootCmd.PersistentFlags().StringVar(&host, "host", "http://localhost", "host where Bytebase is running. e.g. https://bytebase.example.com")
	rootCmd.PersistentFlags().IntVar(&port, "port", 8080, "port where Bytebase is running. e.g. 8080")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data", "./data", "directory where Bytebase stores data.")
}

// -----------------------------------Command Line Config END--------------------------------------

// -----------------------------------Main Entry Point---------------------------------------------
type profile struct {
	mode      string
	logConfig zap.Config
	dsn       string
}

type main struct {
	profile *profile

	l *zap.Logger

	server *server.Server

	db *store.DB
}

func start() {
	m := newMain()

	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		m.l.Info("SIGINT received.")
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

	m.l.Info("Bytebase stopped properly.")
}

func newMain() *main {
	var activeProfile = map[string]profile{
		"dev": {
			mode:      "dev",
			logConfig: zap.NewDevelopmentConfig(),
			dsn:       fmt.Sprintf("file:%s/bytebase_dev.db", dataDir),
		},
		"release": {
			mode:      "release",
			logConfig: zap.NewProductionConfig(),
			dsn:       fmt.Sprintf("file:%s/bytebase.db", dataDir),
		},
		"demo": {
			mode:      "demo",
			logConfig: zap.NewProductionConfig(),
			dsn:       fmt.Sprintf("file:%s/bytebase_demo.db", dataDir),
		},
	}[mode]

	fmt.Println("-----Config BEGIN-----")
	fmt.Printf("mode=%s\n", activeProfile.mode)
	fmt.Printf("host=%s\n", host)
	fmt.Printf("port=%d\n", port)
	fmt.Printf("data=%s\n", activeProfile.dsn)
	fmt.Println("-----Config END-------")

	logger, err := activeProfile.logConfig.Build()
	if err != nil {
		panic(fmt.Errorf("failed to create logger. %w", err))
	}

	defer logger.Sync()
	return &main{
		profile: &activeProfile,
		l:       logger,
	}
}

func (m *main) Run() error {
	db := store.NewDB(m.l, m.profile.dsn, m.profile.mode)
	if err := db.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	m.db = db

	server := server.NewServer(m.l, host, port)
	server.PrincipalService = store.NewPrincipalService(m.l, db)
	server.MemberService = store.NewMemberService(m.l, db)
	server.ProjectService = store.NewProjectService(m.l, db)
	server.ProjectMemberService = store.NewProjectMemberService(m.l, db)
	server.EnvironmentService = store.NewEnvironmentService(m.l, db)
	server.DataSourceService = store.NewDataSourceService(m.l, db)
	server.DatabaseService = store.NewDatabaseService(m.l, db)
	server.InstanceService = store.NewInstanceService(m.l, db, server.DatabaseService, server.DataSourceService)
	server.TableService = store.NewTableService(m.l, db)
	server.IssueService = store.NewIssueService(m.l, db)
	server.PipelineService = store.NewPipelineService(m.l, db)
	server.StageService = store.NewStageService(m.l, db)
	server.TaskService = store.NewTaskService(m.l, db, store.NewTaskRunService(m.l, db))
	server.ActivityService = store.NewActivityService(m.l, db)
	server.BookmarkService = store.NewBookmarkService(m.l, db)
	server.VCSService = store.NewVCSService(m.l, db)
	server.RepositoryService = store.NewRepositoryService(m.l, db, server.ProjectService)

	m.server = server
	if err := server.Run(); err != nil {
		return err
	}

	return nil
}

// Close gracefully stops the program.
func (m *main) Close() error {
	m.l.Info("Trying to stop Bytebase...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if m.server != nil {
		m.l.Info("Trying to gracefully shutdown server...")
		m.server.Shutdown(ctx)
	}

	if m.db != nil {
		m.l.Info("Trying to close database connections...")
		if err := m.db.Close(); err != nil {
			return err
		}
	}
	return nil
}
