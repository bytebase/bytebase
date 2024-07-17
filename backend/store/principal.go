package store

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var systemBotUser = &UserMessage{
	ID:    api.SystemBotID,
	Name:  "Bytebase",
	Email: api.SystemBotEmail,
	Type:  api.SystemBot,
	Roles: []api.Role{api.WorkspaceAdmin},
}

// FindUserMessage is the message for finding users.
type FindUserMessage struct {
	ID          *int
	Email       *string
	Role        *api.Role
	ShowDeleted bool
	Type        *api.PrincipalType
	Limit       *int
}

// UpdateUserMessage is the message to update a user.
type UpdateUserMessage struct {
	Email        *string
	Name         *string
	PasswordHash *string
	Roles        *[]api.Role
	Delete       *bool
	MFAConfig    *storepb.MFAConfig
	Phone        *string
}

// UserMessage is the message for an user.
type UserMessage struct {
	ID int
	// Email must be lower case.
	Email         string
	Name          string
	Type          api.PrincipalType
	PasswordHash  string
	Roles         []api.Role
	MemberDeleted bool
	MFAConfig     *storepb.MFAConfig
	// Phone conforms E.164 format.
	Phone string
}

// GetSystemBotUser gets the system bot.
func (s *Store) GetSystemBotUser(ctx context.Context) *UserMessage {
	user, err := s.GetUserByID(ctx, api.SystemBotID)
	if err != nil {
		slog.Error("failed to find system bot", slog.Int("id", api.SystemBotID), log.BBError(err))
		return systemBotUser
	}
	if user == nil {
		return systemBotUser
	}
	return user
}

// GetUserByID gets the user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int) (*UserMessage, error) {
	if v, ok := s.userIDCache.Get(id); ok {
		return v, nil
	}

	users, err := s.listAndCacheAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, nil
}

// GetUserByEmail gets the user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*UserMessage, error) {
	if v, ok := s.userEmailCache.Get(email); ok {
		return v, nil
	}

	users, err := s.listAndCacheAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

// ListUsers list users.
func (s *Store) ListUsers(ctx context.Context, find *FindUserMessage) ([]*UserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	users, err := s.listUserImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, user := range users {
		s.userIDCache.Add(user.ID, user)
		s.userEmailCache.Add(user.Email, user)
	}
	return users, nil
}

// listAndCacheAllUsers is used for caching all users.
func (s *Store) listAndCacheAllUsers(ctx context.Context) ([]*UserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	users, err := s.listUserImpl(ctx, tx, &FindUserMessage{ShowDeleted: true})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, user := range users {
		s.userIDCache.Add(user.ID, user)
		s.userEmailCache.Add(user.Email, user)
	}
	return users, nil
}

func (s *Store) listUserImpl(ctx context.Context, tx *Tx, find *FindUserMessage) ([]*UserMessage, error) {
	workspaceIamPolicy, err := s.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, err
	}

	var binding *storepb.Binding
	if v := find.Role; v != nil {
		binding = findBindingByRole(workspaceIamPolicy, *v)
	}

	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("principal.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Email; v != nil {
		if *v == api.AllUsers {
			where, args = append(where, fmt.Sprintf("principal.email = $%d", len(args)+1)), append(args, *v)
		} else {
			where, args = append(where, fmt.Sprintf("principal.email = $%d", len(args)+1)), append(args, strings.ToLower(*v))
		}
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("principal.type = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("principal.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	query := `
	SELECT
		principal.id AS user_id,
		principal.row_status AS row_status,
		principal.email,
		principal.name,
		principal.type,
		principal.password_hash,
		principal.mfa_config,
		principal.phone
	FROM principal
	WHERE ` + strings.Join(where, " AND ")

	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

	var userMessages []*UserMessage
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userMessage UserMessage
		var rowStatus string
		var mfaConfigBytes []byte
		if err := rows.Scan(
			&userMessage.ID,
			&rowStatus,
			&userMessage.Email,
			&userMessage.Name,
			&userMessage.Type,
			&userMessage.PasswordHash,
			&mfaConfigBytes,
			&userMessage.Phone,
		); err != nil {
			return nil, err
		}

		if b := binding; b != nil {
			if !slices.Contains(b.Members, common.FormatUserUID(userMessage.ID)) {
				continue
			}
		}

		if userMessage.ID == api.SystemBotID {
			userMessage.Roles = append(userMessage.Roles, api.WorkspaceAdmin)
		} else {
			userMessage.Roles = findRolesByUserID(workspaceIamPolicy, userMessage.ID)
		}

		userMessage.MemberDeleted = convertRowStatusToDeleted(rowStatus)
		mfaConfig := storepb.MFAConfig{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(mfaConfigBytes, &mfaConfig); err != nil {
			return nil, err
		}
		userMessage.MFAConfig = &mfaConfig
		userMessages = append(userMessages, &userMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userMessages, nil
}

// CreateUser creates an user.
func (s *Store) CreateUser(ctx context.Context, create *UserMessage, creatorID int) (*UserMessage, error) {
	// Double check the passing-in emails.
	// We use lower-case for emails.
	if create.Email != strings.ToLower(create.Email) {
		return nil, errors.Errorf("emails must be lower-case when they are passed into store")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	set := []string{"creator_id", "updater_id", "email", "name", "type", "password_hash", "phone"}
	args := []any{creatorID, creatorID, create.Email, create.Name, create.Type, create.PasswordHash, create.Phone}
	placeholder := []string{}
	for index := range set {
		placeholder = append(placeholder, fmt.Sprintf("$%d", index+1))
	}

	var userID int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			INSERT INTO principal (
				%s
			)
			VALUES (%s)
			RETURNING id
		`, strings.Join(set, ","), strings.Join(placeholder, ",")),
		args...,
	).Scan(&userID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	roles := uniq(create.Roles)
	if len(roles) > 0 {
		if _, err := s.UpdateWorkspaceIamPolicy(ctx, &UpdateIamPolicyMessage{
			UserUID:    userID,
			Roles:      roles,
			UpdaterUID: creatorID,
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to update user roles")
		}
	}

	user := &UserMessage{
		ID:           userID,
		Email:        create.Email,
		Name:         create.Name,
		Type:         create.Type,
		PasswordHash: create.PasswordHash,
		Phone:        create.Phone,
		Roles:        roles,
	}
	s.userIDCache.Add(user.ID, user)
	s.userEmailCache.Add(user.Email, user)
	return user, nil
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, currentUser *UserMessage, patch *UpdateUserMessage, updaterID int) (*UserMessage, error) {
	if currentUser.ID == api.SystemBotID {
		return nil, errors.Errorf("cannot update system bot")
	}

	principalSet, principalArgs := []string{"updater_id = $1", "updated_ts = $2"}, []any{updaterID, time.Now().Unix()}
	if v := patch.Delete; v != nil {
		rowStatus := api.Normal
		if *patch.Delete {
			rowStatus = api.Archived
		}
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("row_status = $%d", len(principalArgs)+1)), append(principalArgs, rowStatus)
	}
	if v := patch.Email; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("email = $%d", len(principalArgs)+1)), append(principalArgs, strings.ToLower(*v))
	}
	if v := patch.Name; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("name = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.PasswordHash; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("password_hash = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.Phone; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("phone = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.MFAConfig; v != nil {
		mfaConfigBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("mfa_config = $%d", len(principalArgs)+1)), append(principalArgs, mfaConfigBytes)
	}
	principalArgs = append(principalArgs, currentUser.ID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE principal
		SET `+strings.Join(principalSet, ", ")+`
		WHERE id = $%d
	`, len(principalArgs)),
		principalArgs...,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// patch.Roles overrides patch.Role
	if v := patch.Roles; v != nil {
		if _, err := s.UpdateWorkspaceIamPolicy(ctx, &UpdateIamPolicyMessage{
			UserUID:    currentUser.ID,
			Roles:      *v,
			UpdaterUID: updaterID,
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to update user roles")
		}
	}

	s.userEmailCache.Remove(currentUser.Email)
	s.userIDCache.Remove(currentUser.ID)
	user, err := s.GetUserByID(ctx, currentUser.ID)
	if err != nil {
		return nil, err
	}

	s.userIDCache.Add(currentUser.ID, user)
	s.userEmailCache.Add(user.Email, user)
	return user, nil
}

func uniq[T comparable](array []T) []T {
	res := make([]T, 0, len(array))
	seen := make(map[T]struct{}, len(array))

	for _, e := range array {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		res = append(res, e)
	}

	return res
}
