package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/recovery"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
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
	flags.metricsRemoteAccess = true
	t.Setenv("PG_URL", "postgres://bytebase.example/metadata")

	profile, err := resolveProfile()

	require.NoError(t, err)
	require.Equal(t, dataDir, profile.DataDir)
	require.Equal(t, dataDir, flags.dataDir)
	require.Equal(t, 9090, profile.Port)
	require.Equal(t, 9092, profile.DatastorePort)
	require.True(t, profile.HA)
	require.True(t, profile.MetricsRemoteAccess)
	require.Equal(t, "postgres://bytebase.example/metadata", profile.PgURL)
}

func TestRecoveryRefusesToStartPostgresOnADataDirectoryInUse(t *testing.T) {
	pgDataDir := t.TempDir()
	connectErr := errors.New("the database system is starting up")
	require.NoError(t, refuseRunningEmbeddedPostgres(pgDataDir, connectErr))

	require.NoError(t, os.WriteFile(filepath.Join(pgDataDir, "postmaster.pid"), []byte("12345\n"), 0o600))
	err := refuseRunningEmbeddedPostgres(pgDataDir, connectErr)

	require.ErrorContains(t, err, "the database system is starting up")
	require.ErrorContains(t, err, "may be in use")
	require.ErrorContains(t, err, "inside the Bytebase server's container")
}

// The result lines promise "Live within one minute."
func TestRecoveryPromisesTheStoreCacheLifetime(t *testing.T) {
	require.Equal(t, time.Minute, store.OutOfBandCacheLifetime)
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
	enableResult           *recovery.EnablePasswordSigninResult
	enableErr              error
	resetResult            *recovery.ResetUserPasswordResult
	resetErr               error
	resetErrs              []error
	listErr                error
	addResult              *recovery.AddUserToWorkspaceResult
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
		return f.enableResult, f.enableErr
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
	return &recovery.ResetUserPasswordResult{WorkspaceID: request.WorkspaceID, Email: common.NormalizeEmail(request.Email)}, nil
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
		return f.addResult, f.addErr
	}
	if f.addResult != nil {
		return f.addResult, nil
	}
	return &recovery.AddUserToWorkspaceResult{
		WorkspaceID: request.WorkspaceID,
		Email:       request.Email,
		Role:        request.Role,
		Changed:     true,
	}, nil
}

func TestRecoveryCommandContract(t *testing.T) {
	t.Run("help lists the subcommands and no workspace or confirmation flag", func(t *testing.T) {
		help := recoveryHelp(t)

		require.Contains(t, help[""], "enable-password-signin")
		require.Contains(t, help[""], "reset-password")
		for _, text := range help {
			for _, rejected := range []string{"--workspace", "--yes", "--add-role", "--wait", "--password-file"} {
				require.NotContains(t, text, rejected)
			}
		}
		require.NotContains(t, help["enable-password-signin"], "--email")
		require.NotContains(t, help["enable-password-signin"], "--password")
		require.Contains(t, help["reset-password"], "--email string")
		require.Contains(t, help["reset-password"], "--password string")
	})

	t.Run("rejects a non-terminal before opening metadata and names the subcommands", func(t *testing.T) {
		command, _, openCount, _ := newTestRecoveryCommandWithTerminal(t, &fakeRecoveryRunner{}, false, "", nil)

		err := command.Execute()

		require.ErrorContains(t, err, "terminal input and output")
		require.ErrorContains(t, err, `"bytebase recovery enable-password-signin"`)
		require.ErrorContains(t, err, `"bytebase recovery reset-password --email <email> --password <password>"`)
		require.Zero(t, *openCount)
	})

	t.Run("opens and closes metadata once for the session", func(t *testing.T) {
		command, output, openCount, closeCount := newTestRecoveryCommand(t, &fakeRecoveryRunner{}, "", nil)

		require.NoError(t, command.Execute())
		require.Equal(t, 1, *openCount)
		require.Equal(t, 1, *closeCount)
		requireNoStopOrRestart(t, output.String())
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
		require.Contains(t, output.String(), "Password sign-in is enabled. Live within one minute.")
		require.NotContains(t, output.String(), "resolved-workspace")
		require.Contains(t, output.String(), "Sign in at /auth with an existing user account.")
		require.NotContains(t, output.String(), "existing administrator account")
		require.Less(t, strings.Index(output.String(), "Live within one minute."), strings.Index(output.String(), "Sign in at /auth"))
		requireNoStopOrRestart(t, output.String())
		require.Equal(t, 2, strings.Count(output.String(), "Select a recovery action:"))
	})

	t.Run("enable failure is actionable and returns to the menu", func(t *testing.T) {
		runner := &fakeRecoveryRunner{enableErr: errors.New("no usable workspace administrator")}
		command, output, _, _ := newTestRecoveryCommand(t, runner, "1\ny\nq\n", nil)

		require.NoError(t, command.Execute())
		require.Contains(t, output.String(), "no usable workspace administrator")
		require.Equal(t, 2, strings.Count(output.String(), "Select a recovery action:"))
	})

	t.Run("enable reports a saved setting with a warning when the audit record fails", func(t *testing.T) {
		runner := &fakeRecoveryRunner{
			enableResult: &recovery.EnablePasswordSigninResult{WorkspaceID: "acme", UsableAdminCount: 1},
			enableErr:    errors.New("password sign-in was enabled, but failed to create the recovery audit log: audit write failed"),
		}
		command, output, _, _ := newTestRecoveryCommand(t, runner, "1\ny\nq\n", nil)

		require.NoError(t, command.Execute())
		require.Contains(t, output.String(), "Password sign-in is enabled. Live within one minute.\n"+
			"Warning: password sign-in was enabled, but failed to create the recovery audit log: audit write failed\n")
		require.NotContains(t, output.String(), "Failed to enable password sign-in")
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
		require.Contains(t, output.String(), "Password updated for user user@example.com. Live within one minute.")
		require.NotContains(t, output.String(), "NewPassword1!")
		requireNoStopOrRestart(t, output.String())
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
		require.Contains(t, output.String(), "User user@example.com was added as Workspace admin (roles/workspaceAdmin). Live within one minute.")
		require.Contains(t, output.String(), "Password updated for user user@example.com.")
		require.NotContains(t, output.String(), "NewPassword1!")
		requireNoStopOrRestart(t, output.String())
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

	for _, tc := range []struct {
		name     string
		runner   *fakeRecoveryRunner
		wantLine string
	}{
		{
			name: "role grant reports a stored role with a warning when its audit record fails",
			runner: &fakeRecoveryRunner{
				addResult: &recovery.AddUserToWorkspaceResult{WorkspaceID: "acme", Email: "user@example.com", Role: "roles/workspaceAdmin", Changed: true},
				addErr:    errors.New("user was added to the workspace, but failed to create the recovery audit log: audit write failed"),
			},
			wantLine: "User user@example.com was added as Workspace admin (roles/workspaceAdmin). Live within one minute.\n" +
				"Warning: user was added to the workspace, but failed to create the recovery audit log: audit write failed\n",
		},
		{
			name: "role grant says the user already has a role another server granted meanwhile",
			runner: &fakeRecoveryRunner{
				addResult: &recovery.AddUserToWorkspaceResult{WorkspaceID: "acme", Email: "user@example.com", Role: "roles/workspaceAdmin"},
			},
			wantLine: "User user@example.com already has Workspace admin (roles/workspaceAdmin). Live within one minute.\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.runner.notInWorkspace = true
			tc.runner.roles = []*recovery.Role{{Name: "roles/workspaceAdmin", Title: "Workspace admin"}}
			passwords := [][]byte{[]byte("NewPassword1!"), []byte("NewPassword1!")}
			readCount := 0
			command, output, _, _ := newTestRecoveryCommand(t, tc.runner, "2\nuser@example.com\ny\n1\ny\nq\n", func(io.Reader) ([]byte, error) {
				password := passwords[readCount]
				readCount++
				return password, nil
			})

			require.NoError(t, command.Execute())
			require.Contains(t, output.String(), "Adding user to workspace...\n"+tc.wantLine)
			require.NotContains(t, output.String(), "Failed to add user to workspace")
		})
	}

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

func TestRecoverySubcommands(t *testing.T) {
	resetArgs := []string{"reset-password", "--email", "user@example.com", "--password", "NewPassword1!"}
	const resetResult = "Password updated for user user@example.com. Live within one minute.\n"

	t.Run("enable-password-signin enables sign-in without a terminal", func(t *testing.T) {
		runner := &fakeRecoveryRunner{}
		command, output, openCount, closeCount := newTestRecoverySubcommand(t, runner, "enable-password-signin")

		require.NoError(t, command.Execute())
		require.Equal(t, []string{"acme"}, runner.enableWorkspaces)
		require.Equal(t, "Password sign-in is enabled. Live within one minute.\n", output.String())
		require.Equal(t, 1, *openCount)
		require.Equal(t, 1, *closeCount)
	})

	t.Run("reset-password sets the password without a terminal or a prompt", func(t *testing.T) {
		runner := &fakeRecoveryRunner{}
		command, output, openCount, closeCount := newTestRecoverySubcommand(t, runner, resetArgs...)

		require.NoError(t, command.Execute())
		require.Equal(t, []recovery.ResetUserPasswordRequest{{
			WorkspaceID: "acme",
			Email:       "user@example.com",
			Password:    []byte("NewPassword1!"),
		}}, runner.resetRequests)
		require.Equal(t, resetResult, output.String())
		require.Equal(t, 1, *openCount)
		require.Equal(t, 1, *closeCount)
	})

	for _, password := range []string{"می\u200cخواهمPw1!", "Dev👩\u200d💻Password1!"} {
		t.Run(fmt.Sprintf("reset-password accepts the typeable password %q", password), func(t *testing.T) {
			runner := &fakeRecoveryRunner{}
			command, _, _, _ := newTestRecoverySubcommand(t, runner,
				"reset-password", "--email", "user@example.com", "--password", password)

			require.NoError(t, command.Execute())
			require.Len(t, runner.resetRequests, 1)
		})
	}

	t.Run("reset-password accepts a password that only looks partly like a reference", func(t *testing.T) {
		runner := &fakeRecoveryRunner{}
		command, _, _, _ := newTestRecoverySubcommand(t, runner,
			"reset-password", "--email", "user@example.com", "--password", "$(Str0ng!)")

		require.NoError(t, command.Execute())
		require.Len(t, runner.resetRequests, 1)
		require.Equal(t, []byte("$(Str0ng!)"), runner.resetRequests[0].Password)
	})

	for _, email := range []string{"User@Example.com", "user@example.com\n", " user@example.com "} {
		t.Run(fmt.Sprintf("reset-password reads the email %q as sign-in does", email), func(t *testing.T) {
			runner := &fakeRecoveryRunner{}
			command, output, _, _ := newTestRecoverySubcommand(t, runner,
				"reset-password", "--email", email, "--password", "NewPassword1!")

			require.NoError(t, command.Execute())
			require.Len(t, runner.resetRequests, 1)
			require.Equal(t, resetResult, output.String())
		})
	}

	t.Run("reset-password succeeds with a warning when the workspace role check fails", func(t *testing.T) {
		runner := &fakeRecoveryRunner{membershipErr: errors.New("membership read failed")}
		command, output, _, _ := newTestRecoverySubcommand(t, runner, resetArgs...)

		require.NoError(t, command.Execute())
		require.Equal(t, resetResult+
			"Warning: could not check whether user user@example.com has a workspace role: membership read failed\n", output.String())
	})

	t.Run("reset-password points a user with no workspace role at the menu", func(t *testing.T) {
		runner := &fakeRecoveryRunner{notInWorkspace: true}
		command, output, _, _ := newTestRecoverySubcommand(t, runner, resetArgs...)

		require.NoError(t, command.Execute())
		require.Equal(t, resetResult+
			"User user@example.com has no workspace role. Before they try to sign in, run \"bytebase recovery\" to add one.\n", output.String())
		require.Len(t, runner.resetRequests, 1)
		require.Empty(t, runner.listWorkspaces)
		require.Empty(t, runner.addRequests)
	})

	for _, tc := range []struct {
		name       string
		args       []string
		runner     *fakeRecoveryRunner
		wantErr    string
		wantResult string
	}{
		{
			name:    "enable-password-signin when no administrator has a password",
			args:    []string{"enable-password-signin"},
			runner:  &fakeRecoveryRunner{enableErr: errors.New("no usable workspace administrator has an active end-user identity with a password credential")},
			wantErr: "no usable workspace administrator has an active end-user identity with a password credential",
		},
		{
			name: "enable-password-signin when the audit record fails after the save",
			args: []string{"enable-password-signin"},
			runner: &fakeRecoveryRunner{
				enableResult: &recovery.EnablePasswordSigninResult{WorkspaceID: "acme", UsableAdminCount: 1},
				enableErr:    errors.New("password sign-in was enabled, but failed to create the recovery audit log: audit write failed"),
			},
			wantErr:    "password sign-in was enabled, but failed to create the recovery audit log: audit write failed",
			wantResult: "Password sign-in is enabled. Live within one minute.\n",
		},
		{
			name:    "reset-password for an unknown user",
			args:    resetArgs,
			runner:  &fakeRecoveryRunner{resetErr: errors.New("the user does not exist")},
			wantErr: "the user does not exist",
		},
		{
			name:    "reset-password for a deactivated or non-user account",
			args:    resetArgs,
			runner:  &fakeRecoveryRunner{resetErr: errors.New(`user "user@example.com" is not an active end user`)},
			wantErr: `user "user@example.com" is not an active end user`,
		},
		{
			name:    "reset-password with a password the workspace restriction refuses",
			args:    resetArgs,
			runner:  &fakeRecoveryRunner{resetErr: errors.New("password does not satisfy the workspace restriction: password length should no less than 16 characters")},
			wantErr: "password does not satisfy the workspace restriction: password length should no less than 16 characters",
		},
		{
			name:    "reset-password for an email outside the allowed domains",
			args:    resetArgs,
			runner:  &fakeRecoveryRunner{resetErr: errors.New("the email does not belong to the workspace's allowed domains")},
			wantErr: "the email does not belong to the workspace's allowed domains",
		},
		{
			name: "reset-password when the audit record fails after the save",
			args: resetArgs,
			runner: &fakeRecoveryRunner{
				resetResult: &recovery.ResetUserPasswordResult{WorkspaceID: "acme", Email: "user@example.com", Changed: true},
				resetErr:    errors.New("user password reset completed, but failed to create the recovery audit log: audit write failed"),
			},
			wantErr:    "user password reset completed, but failed to create the recovery audit log: audit write failed",
			wantResult: resetResult,
		},
		{
			name: "reset-password names a missing workspace role before a later step's error",
			args: resetArgs,
			runner: &fakeRecoveryRunner{
				resetResult:    &recovery.ResetUserPasswordResult{WorkspaceID: "acme", Email: "user@example.com", Changed: true},
				resetErr:       errors.New("user password reset completed, but failed to create the recovery audit log: audit write failed"),
				notInWorkspace: true,
			},
			wantErr:    "user password reset completed, but failed to create the recovery audit log: audit write failed",
			wantResult: resetResult + "User user@example.com has no workspace role. Before they try to sign in, run \"bytebase recovery\" to add one.\n",
		},
		{
			name: "reset-password with the password already set when clearing the lockout fails",
			args: resetArgs,
			runner: &fakeRecoveryRunner{
				resetResult: &recovery.ResetUserPasswordResult{WorkspaceID: "acme", Email: "user@example.com"},
				resetErr:    errors.New("failed to clear the login attempt counter: database unavailable"),
			},
			wantErr:    "failed to clear the login attempt counter: database unavailable",
			wantResult: resetResult,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command, output, _, closeCount := newTestRecoverySubcommand(t, tc.runner, tc.args...)

			err := command.Execute()

			require.EqualError(t, err, tc.wantErr)
			require.Equal(t, tc.wantResult+"Error: "+tc.wantErr+"\n", output.String())
			require.Equal(t, 1, *closeCount)
		})
	}

	for _, tc := range []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "reset-password without flags",
			args:    []string{"reset-password"},
			wantErr: "--email and --password need a value",
		},
		{
			name:    "reset-password without a password",
			args:    []string{"reset-password", "--email", "user@example.com"},
			wantErr: "--password needs a value",
		},
		{
			name:    "reset-password without an email",
			args:    []string{"reset-password", "--password", "NewPassword1!"},
			wantErr: "--email needs a value",
		},
		{
			name:    "reset-password with an empty password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", ""},
			wantErr: "--password needs a value",
		},
		{
			name:    "reset-password with an empty email",
			args:    []string{"reset-password", "--email", "", "--password", "NewPassword1!"},
			wantErr: "--email needs a value",
		},
		{
			name:    "reset-password with a value that is not an email address",
			args:    []string{"reset-password", "--email", "not-an-email", "--password", "NewPassword1!"},
			wantErr: "--email is not an email address",
		},
		{
			name:    "reset-password with an unresolved Kubernetes reference as the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "$(NEW_PASSWORD)"},
			wantErr: "--password looks like a variable reference that was not expanded ($NAME, ${NAME} or $(NAME)); in a Job's args, Kubernetes expands only $(NAME), and only for a variable the container defines",
		},
		{
			name:    "reset-password with an unresolved Kubernetes reference to a dashed name as the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "$(new-password)"},
			wantErr: "--password looks like a variable reference that was not expanded ($NAME, ${NAME} or $(NAME)); in a Job's args, Kubernetes expands only $(NAME), and only for a variable the container defines",
		},
		{
			name:    "reset-password with an unresolved Kubernetes reference to a dotted name as the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "$(db.password)"},
			wantErr: "--password looks like a variable reference that was not expanded ($NAME, ${NAME} or $(NAME)); in a Job's args, Kubernetes expands only $(NAME), and only for a variable the container defines",
		},
		{
			name:    "reset-password with an unresolved Kubernetes reference to a name starting with a digit as the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "$(1PASSWORD)"},
			wantErr: "--password looks like a variable reference that was not expanded ($NAME, ${NAME} or $(NAME)); in a Job's args, Kubernetes expands only $(NAME), and only for a variable the container defines",
		},
		{
			name:    "reset-password with an unexpanded braced shell reference as the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "${NEW_PASSWORD}"},
			wantErr: "--password looks like a variable reference that was not expanded ($NAME, ${NAME} or $(NAME)); in a Job's args, Kubernetes expands only $(NAME), and only for a variable the container defines",
		},
		{
			name:    "reset-password with an unexpanded shell reference as the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "$NEW_PASSWORD"},
			wantErr: "--password looks like a variable reference that was not expanded ($NAME, ${NAME} or $(NAME)); in a Job's args, Kubernetes expands only $(NAME), and only for a variable the container defines",
		},
		{
			name:    "reset-password with a trailing newline in the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "NewPassword1!\n"},
			wantErr: "--password contains a character that cannot be typed at sign-in, such as a trailing newline, a byte-order mark or bytes from a file in another encoding",
		},
		{
			name:    "reset-password with a byte-order mark in the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "\ufeffNewPassword1!"},
			wantErr: "--password contains a character that cannot be typed at sign-in, such as a trailing newline, a byte-order mark or bytes from a file in another encoding",
		},
		{
			name:    "reset-password with a zero-width space in the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "NewPassword1!\u200b"},
			wantErr: "--password contains a character that cannot be typed at sign-in, such as a trailing newline, a byte-order mark or bytes from a file in another encoding",
		},
		{
			name:    "reset-password with a replacement character in the password",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "NewPassw\ufffdrd1!"},
			wantErr: "--password contains a character that cannot be typed at sign-in, such as a trailing newline, a byte-order mark or bytes from a file in another encoding",
		},
		{
			name:    "reset-password with a password that is not valid UTF-8",
			args:    []string{"reset-password", "--email", "user@example.com", "--password", "NewPassw\xf6rd1!"},
			wantErr: "--password contains a character that cannot be typed at sign-in, such as a trailing newline, a byte-order mark or bytes from a file in another encoding",
		},
		{
			name:    "reset-password with positional arguments",
			args:    []string{"reset-password", "user@example.com", "NewPassword1!"},
			wantErr: "bytebase recovery reset-password takes no arguments",
		},
		{
			name:    "enable-password-signin with an argument",
			args:    []string{"enable-password-signin", "now"},
			wantErr: "bytebase recovery enable-password-signin takes no arguments",
		},
		{
			name:    "reset-password with an unknown flag",
			args:    []string{"reset-password", "--emial", "user@example.com", "--password", "NewPassword1!"},
			wantErr: "bytebase recovery reset-password was given a flag it does not know or a flag without a valid value; the usage lists its flags",
		},
		{
			name:    "reset-password with a flag that has no value",
			args:    []string{"reset-password", "--email", "user@example.com", "--password"},
			wantErr: "bytebase recovery reset-password was given a flag it does not know or a flag without a valid value; the usage lists its flags",
		},
		{
			name:    "a misspelled subcommand",
			args:    []string{"enable-password-sign-in"},
			wantErr: `bytebase recovery was given an argument or a flag it does not take; to run one action, use "bytebase recovery enable-password-signin" or "bytebase recovery reset-password --email <email> --password <password>"`,
		},
		{
			name:    "a misspelled subcommand with flags",
			args:    []string{"reset-passwd", "--email", "user@example.com", "--password", "NewPassword1!"},
			wantErr: `bytebase recovery was given an argument or a flag it does not take; to run one action, use "bytebase recovery enable-password-signin" or "bytebase recovery reset-password --email <email> --password <password>"`,
		},
	} {
		t.Run(tc.name+" is a usage error that reads nothing", func(t *testing.T) {
			runner := &fakeRecoveryRunner{}
			command, output, openCount, _ := newTestRecoverySubcommand(t, runner, tc.args...)

			err := command.Execute()

			require.EqualError(t, err, tc.wantErr)
			require.Zero(t, *openCount)
			require.True(t, strings.HasPrefix(output.String(), "Error: "+tc.wantErr+"\nUsage:\n"), output.String())
		})
	}

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "a password given without --password", args: []string{"reset-password", "--email", "user@example.com", "S3cretPassword1!"}},
		{name: "a password given as --email", args: []string{"reset-password", "--email", "S3cretPassword1!", "--password", "user@example.com"}},
		{name: "a password that starts with a dash given without --password", args: []string{"reset-password", "--email", "user@example.com", "-S3cretPassword1!"}},
		{name: "a password that starts with two dashes given without --password", args: []string{"reset-password", "--email", "user@example.com", "--S3cretPassword1!"}},
		{name: "--password and its value in one argument", args: []string{"reset-password", "--email", "user@example.com", "--password S3cretPassword1!"}},
		{name: "a single-dash --password", args: []string{"reset-password", "--email", "user@example.com", "-password=S3cretPassword1!"}},
		{name: "a whole command in one argument", args: []string{"reset-password --email user@example.com --password S3cretPassword1!"}},
	} {
		t.Run(tc.name+" is not repeated in the error", func(t *testing.T) {
			command, output, openCount, _ := newTestRecoverySubcommand(t, &fakeRecoveryRunner{}, tc.args...)

			err := command.Execute()

			require.Error(t, err)
			require.NotContains(t, err.Error(), "S3cretPassword1!")
			require.NotContains(t, output.String(), "S3cretPassword1!")
			require.Zero(t, *openCount)
		})
	}

	for _, args := range [][]string{{"enable-password-signin"}, resetArgs} {
		for _, tc := range []struct {
			name    string
			runner  *fakeRecoveryRunner
			wantErr string
		}{
			{
				name:    "no workspace",
				runner:  &fakeRecoveryRunner{noWorkspace: true},
				wantErr: "no active workspace was found in the metadata database",
			},
			{
				name:    "more than one workspace",
				runner:  &fakeRecoveryRunner{workspaceErr: errors.New("found 2 workspaces; recovery requires exactly one workspace")},
				wantErr: "failed to find the recovery workspace: found 2 workspaces; recovery requires exactly one workspace",
			},
		} {
			t.Run(args[0]+" refuses metadata with "+tc.name, func(t *testing.T) {
				command, output, _, closeCount := newTestRecoverySubcommand(t, tc.runner, args...)

				err := command.Execute()

				require.EqualError(t, err, tc.wantErr)
				require.Equal(t, "Error: "+tc.wantErr+"\n", output.String())
				require.Empty(t, tc.runner.enableWorkspaces)
				require.Empty(t, tc.runner.resetRequests)
				require.Equal(t, 1, *closeCount)
			})
		}
	}
}

func requireNoStopOrRestart(t *testing.T, output string) {
	t.Helper()
	require.NotContains(t, strings.ToLower(output), "stop all")
	require.NotContains(t, strings.ToLower(output), "restart")
}

// recoveryHelp returns the help of the recovery command, keyed "", and of each
// of its subcommands, keyed by name.
func recoveryHelp(t *testing.T) map[string]string {
	t.Helper()
	help := map[string]string{}
	for _, name := range []string{"", "enable-password-signin", "reset-password"} {
		args := []string{"--help"}
		if name != "" {
			args = []string{name, "--help"}
		}
		command, output, openCount, _ := newTestRecoverySubcommand(t, &fakeRecoveryRunner{}, args...)
		require.NoError(t, command.Execute())
		require.Zero(t, *openCount)
		help[name] = output.String()
	}
	return help
}

// newTestRecoverySubcommand returns a root command that runs
// `bytebase recovery <args>` the way a Kubernetes Job does: no terminal and
// nothing on standard input. The output is what the operator sees, including
// the errors and usage the root prints.
func newTestRecoverySubcommand(t *testing.T, runner recoveryRunner, args ...string) (*cobra.Command, *bytes.Buffer, *int, *int) {
	t.Helper()
	recoveryCommand, output, openCount, closeCount := newTestRecoveryCommandWithTerminal(t, runner, false, "", nil)
	recoveryCommand.SilenceErrors = false
	root := &cobra.Command{Use: "bytebase"}
	root.AddCommand(recoveryCommand)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs(append([]string{"recovery"}, args...))
	return root, output, openCount, closeCount
}

func newTestRecoveryCommand(
	t *testing.T,
	runner recoveryRunner,
	input string,
	readPassword func(io.Reader) ([]byte, error),
) (*cobra.Command, *bytes.Buffer, *int, *int) {
	t.Helper()
	return newTestRecoveryCommandWithTerminal(t, runner, true, input, readPassword)
}

func newTestRecoveryCommandWithTerminal(
	t *testing.T,
	runner recoveryRunner,
	terminal bool,
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
		isTerminal:   func(io.Reader, io.Writer) bool { return terminal },
		readPassword: readPassword,
	})
	output := &bytes.Buffer{}
	command.SetIn(strings.NewReader(input))
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs(nil)
	command.SilenceErrors = true
	return command, output, &openCount, &closeCount
}
