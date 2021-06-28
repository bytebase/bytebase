package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/store"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// -----------------------------------Command Line Config BEGIN------------------------------------

const (
	SECRET_LENGTH = 32
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=Bytebase
	GREETING_BANNER = `
██████╗ ██╗   ██╗████████╗███████╗██████╗  █████╗ ███████╗███████╗
██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝██╔══██╗██╔══██╗██╔════╝██╔════╝
██████╔╝ ╚████╔╝    ██║   █████╗  ██████╔╝███████║███████╗█████╗  
██╔══██╗  ╚██╔╝     ██║   ██╔══╝  ██╔══██╗██╔══██║╚════██║██╔══╝  
██████╔╝   ██║      ██║   ███████╗██████╔╝██║  ██║███████║███████╗
╚═════╝    ╚═╝      ╚═╝   ╚══════╝╚═════╝ ╚═╝  ╚═╝╚══════╝╚══════╝
                                                                  
%s
___________________________________________________________________________________________

`
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=BYE%20BYE
	BYE_BANNER = `
██████╗ ██╗   ██╗███████╗    ██████╗ ██╗   ██╗███████╗
██╔══██╗╚██╗ ██╔╝██╔════╝    ██╔══██╗╚██╗ ██╔╝██╔════╝
██████╔╝ ╚████╔╝ █████╗      ██████╔╝ ╚████╔╝ █████╗  
██╔══██╗  ╚██╔╝  ██╔══╝      ██╔══██╗  ╚██╔╝  ██╔══╝  
██████╔╝   ██║   ███████╗    ██████╔╝   ██║   ███████╗
╚═════╝    ╚═╝   ╚══════╝    ╚═════╝    ╚═╝   ╚══════╝
                                                      
`
)

var (
	// Used for flags.
	host       string
	port       int
	dataDir    string
	secretFile string
	secret     string
	demo       bool
	debug      bool

	logger *zap.Logger

	rootCmd = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase server",
		Run: func(cmd *cobra.Command, args []string) {
			logConfig := zap.NewProductionConfig()
			// Always set encoding to "console" for now since we do not redirect to file.
			logConfig.Encoding = "console"
			if debug {
				logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
			} else {
				logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
				logConfig.DisableStacktrace = true
			}
			myLogger, err := logConfig.Build()
			if err != nil {
				panic(fmt.Errorf("failed to create logger. %w", err))
			}
			logger = myLogger
			defer logger.Sync()

			if err := preStart(); err != nil {
				logger.Error(err.Error())
				os.Exit(1)
			}

			start()

			fmt.Print(BYE_BANNER)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&host, "host", "http://localhost", "host where Bytebase is running. e.g. https://bytebase.example.com")
	rootCmd.PersistentFlags().IntVar(&port, "port", 8080, "port where Bytebase is running e.g. 8080")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data", ".", "directory where Bytebase stores data.")
	rootCmd.PersistentFlags().StringVar(&secretFile, "secret", "./bytebase_secret", "file path storing the secret to sign the JWT for authentication. If file does not exist, Bytebase will generate a 32 byte random string consisting of numbers and letters.")
	rootCmd.PersistentFlags().BoolVar(&demo, "demo", false, "whether to run in demo mode. Demo mode uses demo data and is read-only.")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "whether to enable debug level logging.")
}

// -----------------------------------Command Line Config END--------------------------------------

// -----------------------------------Main Entry Point---------------------------------------------
type profile struct {
	dsn     string
	seedDir string
}

type main struct {
	profile *profile

	l *zap.Logger

	server *server.Server

	db *store.DB
}

func preStart() error {
	// Trim trailing / in case user supplies
	dir, err := filepath.Abs(strings.TrimRight(dataDir, "/"))
	if err != nil {
		return err
	}
	dataDir = dir

	if _, err := os.Stat(dataDir); err != nil {
		error := fmt.Errorf("unable to access --data %s, %w", dataDir, err)
		return error
	}

	secretFile, err = filepath.Abs(secretFile)
	if err != nil {
		return err
	}
	if _, err := os.Stat(secretFile); err == nil {
		data, err := ioutil.ReadFile(secretFile)
		if err != nil {
			return fmt.Errorf("unable to read secret, error: %w", err)
		}
		secret = string(data)
	} else if errors.Is(err, fs.ErrNotExist) {
		logger.Info(fmt.Sprintf("Secret file does not exist on the specified path %s. Generating a new one...\n", secretFile))
		secret = bytebase.RandomString(SECRET_LENGTH)
		if err := ioutil.WriteFile(secretFile, []byte(secret), 0600); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("unable to save secret, please make sure the parent directory exists: %s", filepath.Dir(secretFile))
			}
			return fmt.Errorf("unable to save secret, error: %w", err)
		}
		logger.Info(fmt.Sprintf("Successfully saved secret at %s\n", secretFile))
	} else {
		return fmt.Errorf("unable to stat secret file: %s, error: %w", secretFile, err)
	}
	return nil
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
		}
		cancel()
	}()

	// Execute program.
	if err := m.Run(); err != nil {
		m.l.Error(err.Error())
		if err != http.ErrServerClosed {
			m.Close()
			cancel()
		}
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}

func newMain() *main {
	activeProfile := activeProfile(dataDir, demo)

	fmt.Println("-----Config BEGIN-----")
	fmt.Printf("host=%s\n", host)
	fmt.Printf("port=%d\n", port)
	fmt.Printf("dsn=%s\n", activeProfile.dsn)
	fmt.Printf("seedDir=%s\n", activeProfile.seedDir)
	fmt.Printf("secret=%s\n", secretFile)
	fmt.Printf("demo=%t\n", demo)
	fmt.Printf("debug=%t\n", debug)
	fmt.Println("-----Config END-------")

	return &main{
		profile: &activeProfile,
		l:       logger,
	}
}

func (m *main) Run() error {
	db := store.NewDB(m.l, m.profile.dsn, m.profile.seedDir)
	if err := db.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	m.db = db

	server := server.NewServer(m.l, host, port, secret, demo, debug)
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

	fmt.Printf(GREETING_BANNER, fmt.Sprintf("Version %s has started at %s:%d", version, host, port))

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
	m.l.Info("Bytebase stopped properly.")
	return nil
}
