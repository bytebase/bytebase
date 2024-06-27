package store

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgtype"
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

func (*Store) listUserImpl(ctx context.Context, tx *Tx, find *FindUserMessage) ([]*UserMessage, error) {
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
	if v := find.Role; v != nil {
		where, args = append(where, fmt.Sprintf("$%d = ANY(member_roles.roles)", len(args)+1)), append(args, *v)
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
		principal.phone,
		member_roles.roles
	FROM principal
	LEFT JOIN LATERAL (SELECT ARRAY(SELECT member.role FROM member WHERE member.principal_id = principal.id ORDER BY member.role)) AS member_roles(roles) ON TRUE
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
		var roles pgtype.TextArray
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
			&roles,
		); err != nil {
			return nil, err
		}

		// pgtype cannot assign []string to []api.Role
		var rolesString []string
		if err := roles.AssignTo(&rolesString); err != nil {
			return nil, errors.Wrapf(err, "failed to scan roles")
		}
		for _, r := range rolesString {
			userMessage.Roles = append(userMessage.Roles, api.Role(r))
		}
		if userMessage.ID == api.SystemBotID {
			userMessage.Roles = append(userMessage.Roles, api.WorkspaceAdmin)
		}

		if !containsWorkspaceRole(userMessage.Roles) {
			userMessage.Roles = append(userMessage.Roles, api.WorkspaceMember)
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

	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM member`,
	).Scan(&count); err != nil {
		return nil, err
	}

	roles := create.Roles
	firstMember := count == 0
	// Grant the member Owner role if there is no existing member.
	if firstMember {
		roles = []api.Role{api.WorkspaceAdmin}
	}
	roles = uniq(roles)

	if len(roles) > 0 {
		if _, err := tx.ExecContext(ctx, `
		INSERT INTO member (
			creator_id,
			updater_id,
			role,
			principal_id
		) SELECT $1, $2, unnest($3::text[]), $4`,
			creatorID, creatorID, roles, userID); err != nil {
			return nil, errors.Wrapf(err, "failed to insert members")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
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

	var patchRoles []api.Role
	doPatchRoles := false
	// patch.Roles overrides patch.Role
	if v := patch.Roles; v != nil {
		doPatchRoles = true
		patchRoles = *v
	}
	if doPatchRoles {
		if err := s.updateUserRoles(ctx, tx, currentUser.ID, patchRoles, updaterID); err != nil {
			return nil, errors.Wrapf(err, "failed to update user roles")
		}
	}

	users, err := s.listUserImpl(ctx, tx, &FindUserMessage{ID: &currentUser.ID, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(users) != 1 {
		return nil, errors.Errorf("failed invalid users, %d users with id %d", len(users), currentUser.ID)
	}
	user := users[0]

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.userEmailCache.Remove(currentUser.Email)
	s.userIDCache.Add(currentUser.ID, user)
	s.userEmailCache.Add(user.Email, user)
	return user, nil
}

func (s *Store) updateUserRoles(ctx context.Context, tx *Tx, userUID int, roles []api.Role, updaterUID int) error {
	oldUser, err := s.listUserImpl(ctx, tx, &FindUserMessage{ID: &userUID})
	if err != nil {
		return err
	}
	if len(oldUser) != 1 {
		return errors.Errorf("expect to get one user with uid %d, got %d", userUID, len(oldUser))
	}
	oldMap, newMap := make(map[api.Role]struct{}), make(map[api.Role]struct{})
	for _, r := range oldUser[0].Roles {
		oldMap[r] = struct{}{}
	}
	for _, r := range roles {
		newMap[r] = struct{}{}
	}
	var remove, add []string
	for r := range oldMap {
		if _, ok := newMap[r]; !ok {
			remove = append(remove, r.String())
		}
	}
	for r := range newMap {
		if _, ok := oldMap[r]; !ok {
			add = append(add, r.String())
		}
	}

	if len(remove) > 0 {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM member
			WHERE principal_id = $1 AND role = ANY($2)
		`, userUID, remove); err != nil {
			return err
		}
	}
	if len(add) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO member (principal_id, role, creator_id, updater_id)
			SELECT $1, unnest($2::text[]), $3, $3
		`, userUID, add, updaterUID); err != nil {
			return err
		}
	}

	return nil
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

func containsWorkspaceRole(roles []api.Role) bool {
	for _, role := range roles {
		if strings.HasPrefix(role.String(), "workspace") {
			return true
		}
	}
	return false
}
