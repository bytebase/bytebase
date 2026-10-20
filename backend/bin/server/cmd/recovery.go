package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/recovery"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/store"
)

// recoveryRunner is the recovery service as the command uses it.
type recoveryRunner interface {
	GetWorkspaceID(context.Context) (string, error)
	EnablePasswordSignin(context.Context, string) (*recovery.EnablePasswordSigninResult, error)
	GetPasswordRestriction(context.Context, string) (*storepb.WorkspaceProfileSetting_PasswordRestriction, error)
	ResetUserPassword(context.Context, recovery.ResetUserPasswordRequest) (*recovery.ResetUserPasswordResult, error)
	IsUserInWorkspace(context.Context, string, string) (bool, error)
	ListRoles(context.Context, string) ([]*recovery.Role, error)
	AddUserToWorkspace(context.Context, recovery.AddUserToWorkspaceRequest) (*recovery.AddUserToWorkspaceResult, error)
}

type recoveryBackend struct {
	runner recoveryRunner
	close  func() error
}

type recoveryCommandDependencies struct {
	open         func(context.Context) (*recoveryBackend, error)
	isTerminal   func(io.Reader, io.Writer) bool
	readPassword func(io.Reader) ([]byte, error)
}

func init() {
	rootCmd.AddCommand(newRecoveryCommand(recoveryCommandDependencies{
		open:         openRecoveryBackend,
		isTerminal:   isRecoveryTerminal,
		readPassword: readRecoveryPassword,
	}))
}

// oneActionCommands are the two forms that run one action without a terminal.
const oneActionCommands = `"bytebase recovery enable-password-signin" or "bytebase recovery reset-password --email <email> --password <password>"`

func newRecoveryCommand(dependencies recoveryCommandDependencies) *cobra.Command {
	command := &cobra.Command{
		Use:   "recovery",
		Short: "Recover password sign-in for a self-hosted deployment",
		Args:  noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			command.SilenceUsage = true
			input := command.InOrStdin()
			output := command.OutOrStdout()
			if !dependencies.isTerminal(input, output) {
				return errors.New("recovery requires terminal input and output; without a terminal, run " + oneActionCommands)
			}
			return runRecoveryAction(command, dependencies, func(ctx context.Context, workspaceID string, runner recoveryRunner) error {
				return runRecoverySession(ctx, workspaceID, input, output, runner, dependencies.readPassword)
			})
		},
	}
	// The subcommands inherit this, so no flag error of theirs quotes the
	// command line either.
	command.SetFlagErrorFunc(redactFlagError)
	command.AddCommand(newEnablePasswordSigninCommand(dependencies), newResetPasswordCommand(dependencies))
	return command
}

func newEnablePasswordSigninCommand(dependencies recoveryCommandDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "enable-password-signin",
		Short: "Enable password sign-in when a workspace administrator has a password",
		Args:  noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			command.SilenceUsage = true
			return runRecoveryAction(command, dependencies, func(ctx context.Context, workspaceID string, runner recoveryRunner) error {
				result, err := runner.EnablePasswordSignin(ctx, workspaceID)
				if result != nil {
					printPasswordSigninEnabled(command.OutOrStdout())
				}
				return err
			})
		},
	}
}

func newResetPasswordCommand(dependencies recoveryCommandDependencies) *cobra.Command {
	var email, password string
	command := &cobra.Command{
		Use:   "reset-password --email <email> --password <password>",
		Short: "Set a new password for an existing, active user",
		Args:  noArgs,
		// A Job fills these flags from Secrets; the checks catch what that gets
		// wrong, before anything is read.
		PreRunE: func(*cobra.Command, []string) error {
			switch {
			case email == "" && password == "":
				return errors.New("--email and --password need a value")
			case email == "":
				return errors.New("--email needs a value")
			case password == "":
				return errors.New("--password needs a value")
			}
			if _, err := recovery.ParseEmail(email); err != nil {
				return errors.New("--email is not an email address")
			}
			if unexpandedReference.MatchString(password) {
				return errors.New("--password looks like a variable reference that was not expanded ($NAME, ${NAME} or $(NAME)); in a Job's args, Kubernetes expands only $(NAME), and only for a variable the container defines")
			}
			if !utf8.ValidString(password) || strings.ContainsFunc(password, cannotBeTyped) {
				return errors.New("--password contains a character that cannot be typed at sign-in, such as a trailing newline, a byte-order mark or bytes from a file in another encoding")
			}
			return nil
		},
		RunE: func(command *cobra.Command, _ []string) error {
			command.SilenceUsage = true
			return runRecoveryAction(command, dependencies, func(ctx context.Context, workspaceID string, runner recoveryRunner) error {
				result, err := runner.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
					WorkspaceID: workspaceID,
					Email:       email,
					Password:    []byte(password),
				})
				if result == nil {
					return err
				}
				output := command.OutOrStdout()
				printPasswordResetSuccess(output, result.Email)
				// The role lookup only informs the operator about the saved
				// password, so it runs even when a later step failed, and its own
				// failure is a warning.
				member, lookupErr := runner.IsUserInWorkspace(ctx, workspaceID, result.Email)
				if lookupErr != nil {
					fmt.Fprintf(command.ErrOrStderr(), "Warning: could not check whether user %s has a workspace role: %v\n", result.Email, lookupErr)
				} else if !member {
					fmt.Fprintf(output, "User %s has no workspace role. Before they try to sign in, run \"bytebase recovery\" to add one.\n", result.Email)
				}
				return err
			})
		},
	}
	command.Flags().StringVar(&email, "email", "", "email of an existing, active user")
	command.Flags().StringVar(&password, "password", "", "the new password; it shows in the process list and the shell history, so change it after signing in")
	return command
}

// noArgs and redactFlagError replace cobra's errors, which quote what was
// typed: a password that lost or garbled its --password flag would be printed
// into the log. For the same reason, no other error here repeats a flag's value.
func noArgs(command *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	if command.HasSubCommands() {
		return subcommandUsageError(command)
	}
	return errors.Errorf("%s takes no arguments", command.CommandPath())
}

func redactFlagError(command *cobra.Command, _ error) error {
	if command.HasSubCommands() {
		return subcommandUsageError(command)
	}
	return errors.Errorf("%s was given a flag it does not know or a flag without a valid value; the usage lists its flags", command.CommandPath())
}

// subcommandUsageError answers what `bytebase recovery` does not take, most
// often a misspelled subcommand, with the forms it does take.
func subcommandUsageError(command *cobra.Command) error {
	return errors.Errorf("%s was given an argument or a flag it does not take; to run one action, use %s", command.CommandPath(), oneActionCommands)
}

// unexpandedReference matches a whole value that is a variable reference left
// in place: $NAME and ${NAME} with a shell name, or $(NAME) with any name a
// Secret or ConfigMap key can hold (letters, digits, '-', '.' and '_'), which
// envFrom turns into variables. Kubernetes passes an undefined $(NAME) through
// unchanged. A value with other characters inside, such as $(Str0ng!), is more
// likely a password than a variable's name.
var unexpandedReference = regexp.MustCompile(`^\$([A-Za-z_][A-Za-z0-9_]*|\{[A-Za-z_][A-Za-z0-9_]*\}|\([-._A-Za-z0-9]+\))$`)

// cannotBeTyped reports a character that a Secret file or a pasted document
// adds and a sign-in form cannot receive: control characters such as a
// trailing newline, line and paragraph separators, a byte-order mark, a
// zero-width space or word joiner, and the replacement character a decoder
// leaves for bytes in another encoding. Joiners that keyboards produce, as in
// Persian text or emoji, are allowed.
func cannotBeTyped(r rune) bool {
	switch r {
	case '\ufeff', '\u200b', '\u2060', utf8.RuneError:
		return true
	default:
		return unicode.IsControl(r) || unicode.In(r, unicode.Zl, unicode.Zp)
	}
}

// runRecoveryAction opens the metadata store, resolves its only workspace and
// runs action against it.
func runRecoveryAction(
	command *cobra.Command,
	dependencies recoveryCommandDependencies,
	action func(ctx context.Context, workspaceID string, runner recoveryRunner) error,
) (retErr error) {
	ctx, stop := signal.NotifyContext(command.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	backend, err := dependencies.open(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to open recovery metadata store")
	}
	defer func() {
		if err := backend.close(); err != nil && retErr == nil {
			retErr = errors.Wrap(err, "failed to close recovery metadata store")
		}
	}()

	workspaceID, err := backend.runner.GetWorkspaceID(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to find the recovery workspace")
	}
	if workspaceID == "" {
		return errors.New("no active workspace was found in the metadata database")
	}
	return action(ctx, workspaceID, backend.runner)
}

func openRecoveryBackend(ctx context.Context) (*recoveryBackend, error) {
	profile, err := resolveProfile()
	if err != nil {
		return nil, err
	}
	if !profile.UseEmbedDB() {
		stores, err := store.New(ctx, profile.PgURL, false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to external metadata database")
		}
		return newRecoveryBackend(stores, nil), nil
	}

	pgURL := fmt.Sprintf("host=%s port=%d user=bb database=bb", postgres.GetPostgresSocketDir(), profile.DatastorePort)
	stores, connectErr := store.New(ctx, pgURL, false)
	if connectErr == nil {
		return newRecoveryBackend(stores, nil), nil
	}

	pgDataDir := filepath.Join(profile.DataDir, "pgdata")
	if _, err := os.Stat(filepath.Join(pgDataDir, "PG_VERSION")); err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Errorf("embedded metadata database is not initialized at %q", pgDataDir)
		}
		return nil, errors.Wrap(err, "failed to inspect embedded metadata database")
	}
	if err := refuseRunningEmbeddedPostgres(pgDataDir, connectErr); err != nil {
		return nil, err
	}
	stopEmbeddedPostgres, err := postgres.StartMetadataInstance(ctx, pgDataDir, profile.DatastorePort, profile.Mode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start embedded metadata database")
	}
	stores, err = store.New(ctx, pgURL, false)
	if err != nil {
		stopEmbeddedPostgres()
		return nil, errors.Wrap(err, "failed to connect to embedded metadata database")
	}
	return newRecoveryBackend(stores, stopEmbeddedPostgres), nil
}

// refuseRunningEmbeddedPostgres stops recovery from starting PostgreSQL on a
// data directory that another postmaster may hold. PostgreSQL's own lock
// cannot see a postmaster in another container, so the two would write the
// same files.
func refuseRunningEmbeddedPostgres(pgDataDir string, connectErr error) error {
	_, err := os.Stat(filepath.Join(pgDataDir, "postmaster.pid"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "failed to inspect embedded metadata database")
	}
	return errors.Errorf("cannot connect to the embedded metadata database (%v), and its data directory %q may be in use: run recovery inside the Bytebase server's container, with the server's --data and --port; if no Bytebase server is running, start it once, then run recovery that way", connectErr, pgDataDir)
}

func newRecoveryBackend(stores *store.Store, stopEmbeddedPostgres func()) *recoveryBackend {
	return &recoveryBackend{
		runner: recovery.NewService(stores),
		close: func() error {
			err := stores.Close()
			if stopEmbeddedPostgres != nil {
				stopEmbeddedPostgres()
			}
			return err
		},
	}
}

func runRecoverySession(
	ctx context.Context,
	workspaceID string,
	input io.Reader,
	output io.Writer,
	runner recoveryRunner,
	readPassword func(io.Reader) ([]byte, error),
) error {
	reader := bufio.NewReader(input)
	for {
		if ctx.Err() != nil {
			return recoverySessionResult(ctx.Err())
		}
		fmt.Fprintln(output, "\nSelect a recovery action:")
		fmt.Fprintln(output, "1. Enable password sign-in")
		fmt.Fprintln(output, "2. Reset user password")
		selection, err := readRecoveryLine(ctx, reader, output, "q. Exit\n> ")
		if err != nil {
			if ctx.Err() != nil {
				return recoverySessionResult(ctx.Err())
			}
			if errors.Is(err, io.EOF) {
				return recoverySessionResult(err)
			}
			return errors.Wrap(err, "failed to read recovery action")
		}

		var actionErr error
		switch strings.ToLower(selection) {
		case "1":
			actionErr = runEnablePasswordSignin(ctx, workspaceID, reader, output, runner)
		case "2":
			actionErr = runResetUserPassword(ctx, workspaceID, input, reader, output, runner, readPassword)
		case "q", "quit", "exit":
			return nil
		default:
			fmt.Fprintln(output, "Invalid selection. Choose 1, 2, or q.")
			continue
		}
		if actionErr != nil {
			if ctx.Err() != nil {
				return recoverySessionResult(ctx.Err())
			}
			if errors.Is(actionErr, io.EOF) {
				return recoverySessionResult(actionErr)
			}
			return actionErr
		}
	}
}

func runEnablePasswordSignin(
	ctx context.Context,
	workspaceID string,
	reader *bufio.Reader,
	output io.Writer,
	runner recoveryRunner,
) error {
	confirmation, err := readRecoveryLine(
		ctx,
		reader,
		output,
		"Enable password sign-in? [y/N] ",
	)
	if err != nil {
		return err
	}
	if !strings.EqualFold(confirmation, "y") && !strings.EqualFold(confirmation, "yes") {
		fmt.Fprintln(output, "Enable password sign-in cancelled.")
		return nil
	}

	fmt.Fprintln(output, "Enabling password sign-in...")
	result, err := runner.EnablePasswordSignin(ctx, workspaceID)
	if result == nil {
		fmt.Fprintf(output, "Failed to enable password sign-in: %v\n", err)
		return nil
	}
	printPasswordSigninEnabled(output)
	if err != nil {
		fmt.Fprintf(output, "Warning: %v\n", err)
	}
	fmt.Fprintln(output, "Sign in at /auth with an existing user account.")
	fmt.Fprintln(output, "After repairing SSO, re-enable Disallow password sign-in.")
	return nil
}

func runResetUserPassword(
	ctx context.Context,
	workspaceID string,
	input io.Reader,
	reader *bufio.Reader,
	output io.Writer,
	runner recoveryRunner,
	readPassword func(io.Reader) ([]byte, error),
) error {
	email, err := readRecoveryLine(ctx, reader, output, "User email (blank to cancel): ")
	if err != nil {
		return err
	}
	if email == "" {
		fmt.Fprintln(output, "Reset user password cancelled.")
		return nil
	}
	restriction, err := runner.GetPasswordRestriction(ctx, workspaceID)
	if err != nil {
		fmt.Fprintf(output, "Failed to get workspace password restriction: %v\n", err)
		return nil
	}
	printPasswordRestriction(output, restriction)

	for {
		fmt.Fprint(output, "New password: ")
		password, err := readPassword(input)
		fmt.Fprintln(output)
		if err != nil {
			clearBytes(password)
			return errors.Wrap(err, "failed to read password")
		}
		if err := common.ValidatePassword(string(password), restriction); err != nil {
			clearBytes(password)
			fmt.Fprintf(output, "Password does not satisfy the workspace restriction: %v. Try again.\n", err)
			continue
		}
		fmt.Fprint(output, "Confirm new password: ")
		confirmation, err := readPassword(input)
		fmt.Fprintln(output)
		if err != nil {
			clearBytes(password)
			clearBytes(confirmation)
			return errors.Wrap(err, "failed to confirm password")
		}
		if !bytes.Equal(password, confirmation) {
			clearBytes(password)
			clearBytes(confirmation)
			fmt.Fprintln(output, "Passwords do not match. Try again.")
			continue
		}

		clearBytes(confirmation)
		actionErr := resetUserPassword(ctx, workspaceID, email, password, reader, output, runner)
		clearBytes(password)
		return actionErr
	}
}

func resetUserPassword(
	ctx context.Context,
	workspaceID string,
	email string,
	password []byte,
	reader *bufio.Reader,
	output io.Writer,
	runner recoveryRunner,
) error {
	fmt.Fprintln(output, "Resetting user password...")
	result, resetErr := runner.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
		WorkspaceID: workspaceID,
		Email:       email,
		Password:    password,
	})
	clearBytes(password)
	if result == nil {
		fmt.Fprintf(output, "Failed to reset user password: %v\n", resetErr)
		return nil
	}
	printPasswordResetSuccess(output, result.Email)
	if resetErr != nil {
		fmt.Fprintf(output, "Warning: %v\n", resetErr)
	}

	member, err := runner.IsUserInWorkspace(ctx, workspaceID, result.Email)
	if err != nil {
		fmt.Fprintf(output, "Failed to check workspace membership: %v\n", err)
		return nil
	}
	if member {
		return nil
	}

	fmt.Fprintf(output, "User %s is not a member of this Bytebase workspace.\n", result.Email)
	confirmation, err := readRecoveryLine(ctx, reader, output, "Add this user to the workspace? [y/N] ")
	if err != nil {
		return err
	}
	if !isRecoveryConfirmed(confirmation) {
		fmt.Fprintln(output, "Add user to workspace cancelled.")
		return nil
	}

	roles, err := runner.ListRoles(ctx, workspaceID)
	if err != nil {
		fmt.Fprintf(output, "Failed to list workspace roles: %v\n", err)
		return nil
	}
	role, cancelled, err := selectRecoveryRole(ctx, reader, output, roles)
	if err != nil {
		return err
	}
	if cancelled {
		fmt.Fprintln(output, "Add user to workspace cancelled.")
		return nil
	}

	confirmation, err = readRecoveryLine(
		ctx,
		reader,
		output,
		fmt.Sprintf(
			"Add %s as %s (%s)? [y/N] ",
			result.Email,
			role.Title,
			role.Name,
		),
	)
	if err != nil {
		return err
	}
	if !isRecoveryConfirmed(confirmation) {
		fmt.Fprintln(output, "Add user to workspace cancelled.")
		return nil
	}

	fmt.Fprintln(output, "Adding user to workspace...")
	addResult, addErr := runner.AddUserToWorkspace(ctx, recovery.AddUserToWorkspaceRequest{
		WorkspaceID: workspaceID,
		Email:       result.Email,
		Role:        role.Name,
	})
	if addResult == nil {
		fmt.Fprintf(output, "Failed to add user to workspace: %v\n", addErr)
		return nil
	}
	granted := "already has"
	if addResult.Changed {
		granted = "was added as"
	}
	fmt.Fprintf(output, "User %s %s %s (%s). Live within one minute.\n", result.Email, granted, role.Title, role.Name)
	if addErr != nil {
		fmt.Fprintf(output, "Warning: %v\n", addErr)
	}
	return nil
}

func printPasswordRestriction(output io.Writer, restriction *storepb.WorkspaceProfileSetting_PasswordRestriction) {
	fmt.Fprintln(output, "Password requirements:")
	hasRequirement := false
	if restriction.GetMinLength() > 0 {
		fmt.Fprintf(output, "- Minimum length: %d characters\n", restriction.GetMinLength())
		hasRequirement = true
	}
	if restriction.GetRequireNumber() {
		fmt.Fprintln(output, "- At least one number")
		hasRequirement = true
	}
	if restriction.GetRequireLetter() {
		fmt.Fprintln(output, "- At least one letter")
		hasRequirement = true
	}
	if restriction.GetRequireUppercaseLetter() {
		fmt.Fprintln(output, "- At least one uppercase letter")
		hasRequirement = true
	}
	if restriction.GetRequireSpecialCharacter() {
		fmt.Fprintln(output, "- At least one special character")
		hasRequirement = true
	}
	if !hasRequirement {
		fmt.Fprintln(output, "- None")
	}
}

func selectRecoveryRole(ctx context.Context, reader *bufio.Reader, output io.Writer, roles []*recovery.Role) (*recovery.Role, bool, error) {
	if len(roles) == 0 {
		fmt.Fprintln(output, "No roles are available in this workspace.")
		return nil, true, nil
	}
	fmt.Fprintln(output, "Select a role:")
	for i, role := range roles {
		fmt.Fprintf(output, "%d. %s (%s)\n", i+1, role.Title, role.Name)
	}
	for {
		selection, err := readRecoveryLine(ctx, reader, output, "q. Cancel\n> ")
		if err != nil {
			return nil, false, err
		}
		if strings.EqualFold(selection, "q") || strings.EqualFold(selection, "quit") || strings.EqualFold(selection, "exit") {
			return nil, true, nil
		}
		index, err := strconv.Atoi(selection)
		if err == nil && index >= 1 && index <= len(roles) {
			return roles[index-1], false, nil
		}
		fmt.Fprintf(output, "Invalid role selection. Choose 1-%d or q.\n", len(roles))
	}
}

func printPasswordSigninEnabled(output io.Writer) {
	fmt.Fprintln(output, "Password sign-in is enabled. Live within one minute.")
}

func printPasswordResetSuccess(output io.Writer, email string) {
	fmt.Fprintf(output, "Password updated for user %s. Live within one minute.\n", email)
}

func isRecoveryConfirmed(value string) bool {
	return strings.EqualFold(value, "y") || strings.EqualFold(value, "yes")
}

type recoveryLineResult struct {
	line string
	err  error
}

func readRecoveryLine(ctx context.Context, reader *bufio.Reader, output io.Writer, prompt string) (string, error) {
	fmt.Fprint(output, prompt)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	result := make(chan recoveryLineResult, 1)
	go func() {
		line, err := reader.ReadString('\n')
		result <- recoveryLineResult{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-result:
		if result.err != nil && (!errors.Is(result.err, io.EOF) || len(result.line) == 0) {
			return "", result.err
		}
		return strings.TrimSpace(result.line), nil
	}
}

func recoverySessionResult(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func isRecoveryTerminal(input io.Reader, output io.Writer) bool {
	inputFile, inputOK := input.(*os.File)
	outputFile, outputOK := output.(*os.File)
	return inputOK && outputOK && term.IsTerminal(int(inputFile.Fd())) && term.IsTerminal(int(outputFile.Fd()))
}

func readRecoveryPassword(input io.Reader) (password []byte, retErr error) {
	inputFile, ok := input.(*os.File)
	if !ok {
		return nil, errors.New("password input is not a terminal")
	}
	fd := int(inputFile.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to enter raw terminal mode")
	}
	defer func() {
		if err := term.Restore(fd, state); err != nil && retErr == nil {
			retErr = errors.Wrap(err, "failed to restore terminal state")
		}
	}()

	terminal := struct {
		io.Reader
		io.Writer
	}{
		Reader: inputFile,
		Writer: io.Discard,
	}
	return readRecoveryPasswordInput(terminal)
}

func readRecoveryPasswordInput(input io.ReadWriter) ([]byte, error) {
	password, err := term.NewTerminal(input, "").ReadPassword("")
	return []byte(password), err
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
