package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/recovery"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestResolveProfileUsesServerDataConfiguration(t *testing.T) {
	originalFlags := flags
	originalArgs := os.Args
	t.Cleanup(func() {
		flags = originalFlags
		os.Args = originalArgs
	})

	binDir := t.TempDir()
	dataDir := filepath.Join(binDir, "metadata")
	require.NoError(t, os.Mkdir(dataDir, 0o700))
	os.Args = []string{filepath.Join(binDir, "bytebase")}
	flags.dataDir = "metadata"
	flags.port = 9090
	flags.ha = true
	t.Setenv("PG_URL", "postgres://bytebase.example/metadata")

	profile, err := resolveProfile()

	require.NoError(t, err)
	require.Equal(t, dataDir, profile.DataDir)
	require.Equal(t, dataDir, flags.dataDir)
	require.Equal(t, 9090, profile.Port)
	require.Equal(t, 9092, profile.DatastorePort)
	require.True(t, profile.HA)
	require.Equal(t, "postgres://bytebase.example/metadata", profile.PgURL)
}

func TestReadRecoveryPasswordInputInterrupt(t *testing.T) {
	terminal := struct {
		io.Reader
		io.Writer
	}{
		Reader: bytes.NewReader([]byte{3}),
		Writer: io.Discard,
	}

	password, err := readRecoveryPasswordInput(terminal)

	require.ErrorIs(t, err, io.EOF)
	require.Empty(t, password)
}

type blockingRecoveryReader struct {
	started chan struct{}
	release chan struct{}
}

func (r *blockingRecoveryReader) Read([]byte) (int, error) {
	close(r.started)
	<-r.release
	return 0, io.EOF
}

type fakeRecoveryRunner struct {
	workspaceID            string
	workspaceErr           error
	noWorkspace            bool
	enableWorkspaces       []string
	resetRequests          []recovery.ResetUserPasswordRequest
	listWorkspaces         []string
	addRequests            []recovery.AddUserToWorkspaceRequest
	roles                  []*recovery.Role
	enableErr              error
	resetResult            *recovery.ResetUserPasswordResult
	resetErr               error
	resetErrs              []error
	listErr                error
	addErr                 error
	passwordRestriction    *storepb.WorkspaceProfileSetting_PasswordRestriction
	passwordRestrictionErr error
	restrictionWorkspaces  []string
	notInWorkspace         bool
	membershipErr          error
	membershipChecks       []string
}

func (f *fakeRecoveryRunner) GetWorkspaceID(context.Context) (string, error) {
	if f.workspaceErr != nil {
		return "", f.workspaceErr
	}
	if f.noWorkspace {
		return "", nil
	}
	if f.workspaceID == "" {
		return "acme", nil
	}
	return f.workspaceID, nil
}

func (f *fakeRecoveryRunner) EnablePasswordSignin(_ context.Context, workspaceID string) (*recovery.EnablePasswordSigninResult, error) {
	f.enableWorkspaces = append(f.enableWorkspaces, workspaceID)
	if f.enableErr != nil {
		return nil, f.enableErr
	}
	return &recovery.EnablePasswordSigninResult{WorkspaceID: workspaceID}, nil
}

func (f *fakeRecoveryRunner) GetPasswordRestriction(_ context.Context, workspaceID string) (*storepb.WorkspaceProfileSetting_PasswordRestriction, error) {
	f.restrictionWorkspaces = append(f.restrictionWorkspaces, workspaceID)
	if f.passwordRestrictionErr != nil {
		return nil, f.passwordRestrictionErr
	}
	if f.passwordRestriction == nil {
		return &storepb.WorkspaceProfileSetting_PasswordRestriction{MinLength: 8}, nil
	}
	return f.passwordRestriction, nil
}

func (f *fakeRecoveryRunner) IsUserInWorkspace(_ context.Context, workspaceID, email string) (bool, error) {
	f.membershipChecks = append(f.membershipChecks, workspaceID+":"+email)
	if f.membershipErr != nil {
		return false, f.membershipErr
	}
	return !f.notInWorkspace, nil
}

func (f *fakeRecoveryRunner) ResetUserPassword(_ context.Context, request recovery.ResetUserPasswordRequest) (*recovery.ResetUserPasswordResult, error) {
	f.resetRequests = append(f.resetRequests, recovery.ResetUserPasswordRequest{
		WorkspaceID: request.WorkspaceID,
		Email:       request.Email,
		Password:    bytes.Clone(request.Password),
	})
	requestIndex := len(f.resetRequests) - 1
	if requestIndex < len(f.resetErrs) && f.resetErrs[requestIndex] != nil {
		return nil, f.resetErrs[requestIndex]
	}
	if f.resetErr != nil {
		return f.resetResult, f.resetErr
	}
	if f.resetResult != nil {
		return f.resetResult, nil
	}
	return &recovery.ResetUserPasswordResult{WorkspaceID: request.WorkspaceID, Email: request.Email}, nil
}

func (f *fakeRecoveryRunner) ListRoles(_ context.Context, workspaceID string) ([]*recovery.Role, error) {
	f.listWorkspaces = append(f.listWorkspaces, workspaceID)
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.roles, nil
}

func (f *fakeRecoveryRunner) AddUserToWorkspace(_ context.Context, request recovery.AddUserToWorkspaceRequest) (*recovery.AddUserToWorkspaceResult, error) {
	f.addRequests = append(f.addRequests, request)
	if f.addErr != nil {
		return nil, f.addErr
	}
	return &recovery.AddUserToWorkspaceResult{
		WorkspaceID: request.WorkspaceID,
		Email:       request.Email,
		Role:        request.Role,
		Changed:     true,
	}, nil
}

func TestRecoveryCommandContract(t *testing.T) {
	t.Run("help exposes no workspace or action inputs", func(t *testing.T) {
		command, output, _, _ := newTestRecoveryCommand(t, &fakeRecoveryRunner{}, "", nil)
		command.SetArgs([]string{"--help"})

		require.NoError(t, command.Execute())
		require.NotContains(t, output.String(), "--workspace")
		require.NotContains(t, output.String(), "--email")
		require.NotContains(t, output.String(), "--password")
		require.NotContains(t, output.String(), "--yes")
		require.Empty(t, command.Commands())
	})

	t.Run("rejects a non-terminal before opening metadata", func(t *testing.T) {
		openCount := 0
		command := newRecoveryCommand(recoveryCommandDependencies{
			open: func(context.Context) (*recoveryBackend, error) {
				openCount++
				return nil, nil
			},
			isTerminal: func(io.Reader, io.Writer) bool { return false },
		})
		command.SetArgs(nil)
		command.SilenceUsage = true
		command.SilenceErrors = true

		err := command.Execute()

		require.ErrorContains(t, err, "terminal input and output")
		require.Zero(t, openCount)
	})

	t.Run("opens and closes metadata once for the session", func(t *testing.T) {
		command, output, openCount, closeCount := newTestRecoveryCommand(t, &fakeRecoveryRunner{}, "", nil)

		require.NoError(t, command.Execute())
		require.Equal(t, 1, *openCount)
		require.Equal(t, 1, *closeCount)
		require.Contains(t, output.String(), "Stop all Bytebase servers that use this metadata database before continuing.")
		require.Contains(t, output.String(), "Restart all Bytebase servers after recovery")
	})

	t.Run("rejects metadata with multiple workspaces", func(t *testing.T) {
		runner := &fakeRecoveryRunner{workspaceErr: errors.New("multiple workspaces were found")}
		command, _, openCount, closeCount := newTestRecoveryCommand(t, runner, "q\n", nil)

		err := command.Execute()

		require.ErrorContains(t, err, "multiple workspaces")
		require.Equal(t, 1, *openCount)
		require.Equal(t, 1, *closeCount)
	})

	t.Run("rejects metadata without an active workspace", func(t *testing.T) {
		runner := &fakeRecoveryRunner{noWorkspace: true}
		command, _, openCount, closeCount := newTestRecoveryCommand(t, runner, "q\n", nil)

		err := command.Execute()

		require.ErrorContains(t, err, "no active workspace")
		require.Equal(t, 1, *openCount)
		require.Equal(t, 1, *closeCount)
	})

	t.Run("invalid selection returns to the menu", func(t *testing.T) {
		command, output, _, _ := newTestRecoveryCommand(t, &fakeRecoveryRunner{}, "invalid\nq\n", nil)

		require.NoError(t, command.Execute())
		require.Contains(t, output.String(), "Invalid selection")
		require.Equal(t, 2, strings.Count(output.String(), "Select a recovery action:"))
	})

	t.Run("cancelled enable does not call the service", func(t *testing.T) {
		runner := &fakeRecoveryRunner{}
		command, _, _, _ := newTestRecoveryCommand(t, runner, "1\nn\nq\n", nil)

		require.NoError(t, command.Execute())
		require.Empty(t, runner.enableWorkspaces)
	})

	t.Run("enable reports progress and returns to the menu", func(t *testing.T) {
		runner := &fakeRecoveryRunner{workspaceID: "resolved-workspace"}
		command, output, _, _ := newTestRecoveryCommand(t, runner, "1\ny\nq\n", nil)

		require.NoError(t, command.Execute())
		require.Equal(t, []string{"resolved-workspace"}, runner.enableWorkspaces)
		require.Contains(t, output.String(), "Enabling password sign-in...")
		require.Contains(t, output.String(), "Password sign-in is enabled.")
		require.NotContains(t, output.String(), "resolved-workspace")
		require.Contains(t, output.String(), "Restart all Bytebase servers after exiting recovery.")
		require.Contains(t, output.String(), "After restarting, sign in at /auth with an existing user account.")
		require.NotContains(t, output.String(), "existing administrator account")
		require.Less(t, strings.Index(output.String(), "Restart all Bytebase servers"), strings.Index(output.String(), "After restarting, sign in at /auth"))
		require.Equal(t, 2, strings.Count(output.String(), "Select a recovery action:"))
	})

	t.Run("enable failure is actionable and returns to the menu", func(t *testing.T) {
		runner := &fakeRecoveryRunner{enableErr: errors.New("no usable workspace administrator")}
		command, output, _, _ := newTestRecoveryCommand(t, runner, "1\ny\nq\n", nil)

		require.NoError(t, command.Execute())
		require.Contains(t, output.String(), "no usable workspace administrator")
		require.Equal(t, 2, strings.Count(output.String(), "Select a recovery action:"))
	})

	t.Run("blank reset email returns to the menu", func(t *testing.T) {
		runner := &fakeRecoveryRunner{}
		command, _, _, _ := newTestRecoveryCommand(t, runner, "2\n\nq\n", nil)

		require.NoError(t, command.Execute())
		require.Empty(t, runner.resetRequests)
		require.Empty(t, runner.restrictionWorkspaces)
	})

	t.Run("password mismatch re-prompts without calling the service", func(t *testing.T) {
		runner := &fakeRecoveryRunner{}
		passwords := [][]byte{[]byte("FirstPassword1!"), []byte("DifferentPassword1!")}
		readCount := 0
		stop := errors.New("stop after mismatch")
		command, output, _, _ := newTestRecoveryCommand(t, runner, "2\nuser@example.com\n", func(io.Reader) ([]byte, error) {
			if readCount == len(passwords) {
				return nil, stop
			}
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		err := command.Execute()

		require.ErrorIs(t, err, stop)
		require.Empty(t, runner.resetRequests)
		require.Equal(t, 2, readCount)
		require.Contains(t, output.String(), "Passwords do not match")
	})

	for _, tc := range []struct {
		name          string
		readPassword  func(io.Reader) ([]byte, error)
		wantReadCount int
	}{
		{
			name: "interrupt at new password exits the session",
			readPassword: func(io.Reader) ([]byte, error) {
				return nil, io.EOF
			},
			wantReadCount: 1,
		},
		{
			name: "interrupt at confirmation exits the session",
			readPassword: func() func(io.Reader) ([]byte, error) {
				readCount := 0
				return func(io.Reader) ([]byte, error) {
					readCount++
					if readCount == 1 {
						return []byte("NewPassword1!"), nil
					}
					return nil, io.EOF
				}
			}(),
			wantReadCount: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readCount := 0
			runner := &fakeRecoveryRunner{}
			command, _, _, closeCount := newTestRecoveryCommand(t, runner, "2\nuser@example.com\n", func(input io.Reader) ([]byte, error) {
				readCount++
				return tc.readPassword(input)
			})

			require.NoError(t, command.Execute())
			require.Empty(t, runner.resetRequests)
			require.Equal(t, tc.wantReadCount, readCount)
			require.Equal(t, 1, *closeCount)
		})
	}

	t.Run("prints the restriction and re-prompts after password validation fails", func(t *testing.T) {
		runner := &fakeRecoveryRunner{passwordRestriction: &storepb.WorkspaceProfileSetting_PasswordRestriction{
			MinLength:               12,
			RequireNumber:           true,
			RequireLetter:           true,
			RequireUppercaseLetter:  true,
			RequireSpecialCharacter: true,
		}}
		passwords := [][]byte{
			[]byte("too-short"),
			[]byte("NewPassword1!"), []byte("NewPassword1!"),
		}
		readCount := 0
		command, output, _, _ := newTestRecoveryCommand(t, runner, "2\nuser@example.com\nq\n", func(io.Reader) ([]byte, error) {
			if readCount == len(passwords) {
				return nil, errors.New("unexpected extra password prompt")
			}
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		require.NoError(t, command.Execute())
		require.Len(t, runner.resetRequests, 1)
		require.Equal(t, []byte("NewPassword1!"), runner.resetRequests[0].Password)
		require.Contains(t, output.String(), "Password requirements:")
		require.Contains(t, output.String(), "Minimum length: 12 characters")
		require.Contains(t, output.String(), "At least one number")
		require.Contains(t, output.String(), "At least one letter")
		require.Contains(t, output.String(), "At least one uppercase letter")
		require.Contains(t, output.String(), "At least one special character")
		require.Contains(t, output.String(), "password length should no less than 12 characters")
		require.Less(t, strings.Index(output.String(), "User email"), strings.Index(output.String(), "Password requirements:"))
		require.Less(t, strings.Index(output.String(), "password length should no less than 12 characters"), strings.Index(output.String(), "Confirm new password:"))
		require.Equal(t, 1, strings.Count(output.String(), "Confirm new password:"))
		require.Equal(t, []string{"acme"}, runner.restrictionWorkspaces)
		require.Equal(t, 3, readCount)
	})

	t.Run("reset passes the email and password without exposing the secret", func(t *testing.T) {
		runner := &fakeRecoveryRunner{}
		firstPassword := []byte("NewPassword1!")
		secondPassword := []byte("NewPassword1!")
		passwords := [][]byte{firstPassword, secondPassword}
		readCount := 0
		command, output, _, _ := newTestRecoveryCommand(t, runner, "2\nuser@example.com\nq\n", func(io.Reader) ([]byte, error) {
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		require.NoError(t, command.Execute())
		require.Len(t, runner.resetRequests, 1)
		require.Equal(t, "acme", runner.resetRequests[0].WorkspaceID)
		require.Equal(t, "user@example.com", runner.resetRequests[0].Email)
		require.Equal(t, []byte("NewPassword1!"), runner.resetRequests[0].Password)
		require.Equal(t, make([]byte, len(firstPassword)), firstPassword)
		require.Equal(t, make([]byte, len(secondPassword)), secondPassword)
		require.Contains(t, output.String(), "Resetting user password...")
		require.Contains(t, output.String(), "Password updated for user user@example.com.")
		require.NotContains(t, output.String(), "NewPassword1!")
	})

	t.Run("missing member can decline the IAM grant", func(t *testing.T) {
		runner := &fakeRecoveryRunner{notInWorkspace: true}
		firstPassword := []byte("NewPassword1!")
		secondPassword := []byte("NewPassword1!")
		passwords := [][]byte{firstPassword, secondPassword}
		readCount := 0
		command, output, _, _ := newTestRecoveryCommand(t, runner, "2\nuser@example.com\nn\nq\n", func(io.Reader) ([]byte, error) {
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		require.NoError(t, command.Execute())
		require.Len(t, runner.resetRequests, 1)
		require.Equal(t, []string{"acme:user@example.com"}, runner.membershipChecks)
		require.Empty(t, runner.listWorkspaces)
		require.Empty(t, runner.addRequests)
		require.Contains(t, output.String(), "User user@example.com is not a member of this Bytebase workspace.")
		require.Contains(t, output.String(), "Add this user to the workspace? [y/N]")
		require.Contains(t, output.String(), "Password updated for user user@example.com.")
		require.Equal(t, make([]byte, len(firstPassword)), firstPassword)
		require.Equal(t, make([]byte, len(secondPassword)), secondPassword)
	})

	t.Run("missing member selects a role after the reset", func(t *testing.T) {
		runner := &fakeRecoveryRunner{
			notInWorkspace: true,
			roles: []*recovery.Role{
				{Name: "roles/workspaceAdmin", Title: "Workspace admin"},
				{Name: "roles/workspaceMember", Title: "Workspace member"},
			},
		}
		firstPassword := []byte("NewPassword1!")
		secondPassword := []byte("NewPassword1!")
		passwords := [][]byte{firstPassword, secondPassword}
		readCount := 0
		command, output, _, _ := newTestRecoveryCommand(t, runner, "2\nuser@example.com\ny\n9\n1\ny\nq\n", func(io.Reader) ([]byte, error) {
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		require.NoError(t, command.Execute())
		require.Equal(t, []string{"acme"}, runner.listWorkspaces)
		require.Equal(t, []recovery.AddUserToWorkspaceRequest{{
			WorkspaceID: "acme",
			Email:       "user@example.com",
			Role:        "roles/workspaceAdmin",
		}}, runner.addRequests)
		require.Len(t, runner.resetRequests, 1)
		require.Equal(t, []byte("NewPassword1!"), runner.resetRequests[0].Password)
		require.Contains(t, output.String(), "1. Workspace admin (roles/workspaceAdmin)")
		require.Contains(t, output.String(), "Invalid role selection")
		require.Contains(t, output.String(), "Add user@example.com as Workspace admin (roles/workspaceAdmin)? [y/N]")
		require.Contains(t, output.String(), "Adding user to workspace...")
		require.Contains(t, output.String(), "Restart all Bytebase servers after exiting recovery so they reload the updated IAM policy.")
		require.Contains(t, output.String(), "Password updated for user user@example.com.")
		require.NotContains(t, output.String(), "NewPassword1!")
		require.Equal(t, make([]byte, len(firstPassword)), firstPassword)
		require.Equal(t, make([]byte, len(secondPassword)), secondPassword)
	})

	t.Run("grant failure does not retry the reset", func(t *testing.T) {
		runner := &fakeRecoveryRunner{
			notInWorkspace: true,
			roles:          []*recovery.Role{{Name: "roles/workspaceMember", Title: "Workspace member"}},
			addErr:         errors.New("IAM write failed"),
		}
		passwords := [][]byte{[]byte("NewPassword1!"), []byte("NewPassword1!")}
		readCount := 0
		command, output, _, _ := newTestRecoveryCommand(t, runner, "2\nuser@example.com\ny\n1\ny\nq\n", func(io.Reader) ([]byte, error) {
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		require.NoError(t, command.Execute())
		require.Len(t, runner.addRequests, 1)
		require.Len(t, runner.resetRequests, 1)
		require.Contains(t, output.String(), "Failed to add user to workspace: IAM write failed")
	})

	t.Run("audit failure preserves the reset and continues membership recovery", func(t *testing.T) {
		runner := &fakeRecoveryRunner{
			resetResult: &recovery.ResetUserPasswordResult{
				WorkspaceID: "acme",
				Email:       "user@example.com",
				Changed:     true,
			},
			resetErr:       errors.New("user password reset completed, but failed to create the recovery audit log: audit write failed"),
			notInWorkspace: true,
			roles:          []*recovery.Role{{Name: "roles/workspaceMember", Title: "Workspace member"}},
		}
		passwords := [][]byte{[]byte("NewPassword1!"), []byte("NewPassword1!")}
		readCount := 0
		command, output, _, _ := newTestRecoveryCommand(t, runner, "2\nuser@example.com\ny\n1\ny\nq\n", func(io.Reader) ([]byte, error) {
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		require.NoError(t, command.Execute())
		require.Equal(t, []string{"acme:user@example.com"}, runner.membershipChecks)
		require.Len(t, runner.addRequests, 1)
		require.Contains(t, output.String(), "Password updated for user user@example.com.")
		require.Contains(t, output.String(), "Warning: user password reset completed, but failed to create the recovery audit log: audit write failed")
	})

	t.Run("membership check failure reports the persisted password reset", func(t *testing.T) {
		runner := &fakeRecoveryRunner{
			membershipErr: errors.New("membership read failed"),
		}
		passwords := [][]byte{[]byte("NewPassword1!"), []byte("NewPassword1!")}
		readCount := 0
		command, output, _, _ := newTestRecoveryCommand(t, runner, "2\nuser@example.com\nq\n", func(io.Reader) ([]byte, error) {
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		require.NoError(t, command.Execute())
		require.Len(t, runner.resetRequests, 1)
		require.Empty(t, runner.addRequests)
		require.Contains(t, output.String(), "Password updated for user user@example.com.")
		require.Contains(t, output.String(), "Failed to check workspace membership: membership read failed")
	})

	t.Run("multiple actions reuse the session", func(t *testing.T) {
		runner := &fakeRecoveryRunner{}
		passwords := [][]byte{[]byte("NewPassword1!"), []byte("NewPassword1!")}
		readCount := 0
		command, _, openCount, closeCount := newTestRecoveryCommand(t, runner, "1\ny\n2\nuser@example.com\nq\n", func(io.Reader) ([]byte, error) {
			password := passwords[readCount]
			readCount++
			return password, nil
		})

		require.NoError(t, command.Execute())
		require.Len(t, runner.enableWorkspaces, 1)
		require.Len(t, runner.resetRequests, 1)
		require.Equal(t, 1, *openCount)
		require.Equal(t, 1, *closeCount)
	})

	t.Run("cancelled context exits cleanly", func(t *testing.T) {
		command, _, _, closeCount := newTestRecoveryCommand(t, &fakeRecoveryRunner{}, "", nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		require.NoError(t, command.ExecuteContext(ctx))
		require.Equal(t, 1, *closeCount)
	})

	t.Run("cancelled context interrupts the menu read", func(t *testing.T) {
		command, _, _, closeCount := newTestRecoveryCommand(t, &fakeRecoveryRunner{}, "", nil)
		input := &blockingRecoveryReader{started: make(chan struct{}), release: make(chan struct{})}
		command.SetIn(input)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			done <- command.ExecuteContext(ctx)
		}()

		<-input.started
		cancel()
		select {
		case err := <-done:
			close(input.release)
			require.NoError(t, err)
			require.Equal(t, 1, *closeCount)
		case <-time.After(time.Second):
			close(input.release)
			<-done
			t.Fatal("recovery did not exit while the menu read was blocked")
		}
	})
}

func newTestRecoveryCommand(
	t *testing.T,
	runner recoveryRunner,
	input string,
	readPassword func(io.Reader) ([]byte, error),
) (*cobra.Command, *bytes.Buffer, *int, *int) {
	t.Helper()
	openCount := 0
	closeCount := 0
	if readPassword == nil {
		readPassword = func(io.Reader) ([]byte, error) {
			return nil, errors.New("unexpected password prompt")
		}
	}
	command := newRecoveryCommand(recoveryCommandDependencies{
		open: func(context.Context) (*recoveryBackend, error) {
			openCount++
			return &recoveryBackend{
				runner: runner,
				close: func() error {
					closeCount++
					return nil
				},
			}, nil
		},
		isTerminal:   func(io.Reader, io.Writer) bool { return true },
		readPassword: readPassword,
	})
	output := &bytes.Buffer{}
	command.SetIn(strings.NewReader(input))
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs(nil)
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command, output, &openCount, &closeCount
}
