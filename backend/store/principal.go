package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SystemBotID is the ID of the system robot.
const SystemBotID = 1

// GetPrincipalByID gets an instance of Principal by ID.
func (s *Store) GetPrincipalByID(ctx context.Context, id int) (*api.Principal, error) {
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	composedPrincipal, err := composePrincipal(user)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Principal role with user [%+v]", user)
	}

	return composedPrincipal, nil
}

// composePrincipal composes an instance of Principal by principalRaw.
func composePrincipal(user *UserMessage) (*api.Principal, error) {
	principal := &api.Principal{
		ID:    user.ID,
		Type:  user.Type,
		Name:  user.Name,
		Email: user.Email,
		// Do not return to the client.
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
	}
	// TODO(d): move this user v1 store.
	if principal.ID == api.SystemBotID {
		principal.Role = api.Owner
	}
	return principal, nil
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
	Role         *api.Role
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
	Role          api.Role
	MemberDeleted bool
	MFAConfig     *storepb.MFAConfig
	// Phone conforms E.164 format.
	Phone string
}

// GetUser gets an user.
func (s *Store) GetUser(ctx context.Context, find *FindUserMessage) (*UserMessage, error) {
	if find.Email != nil && *find.Email == api.SystemBotEmail {
		return &UserMessage{
			ID:    api.SystemBotID,
			Email: api.SystemBotEmail,
			Type:  api.SystemBot,
			Role:  api.Owner,
		}, nil
	}
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

	if len(users) == 0 {
		return nil, nil
	} else if len(users) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d users with filter %+v, expect 1", len(users), find)}
	}
	return users[0], nil
}

// ListUsers list all users.
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
		s.userIDCache.Store(user.ID, user)
	}
	return users, nil
}

// GetUserByID gets the user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int) (*UserMessage, error) {
	if user, ok := s.userIDCache.Load(id); ok {
		return user.(*UserMessage), nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	users, err := s.listUserImpl(ctx, tx, &FindUserMessage{ID: &id, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	if len(users) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d users with id %q, expect 1", len(users), id)}
	}
	user := users[0]
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.userIDCache.Store(user.ID, user)
	return user, nil
}

func (*Store) listUserImpl(ctx context.Context, tx *Tx, find *FindUserMessage) ([]*UserMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("principal.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Email; v != nil {
		where, args = append(where, fmt.Sprintf("principal.email = $%d", len(args)+1)), append(args, strings.ToLower(*v))
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("principal.type = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Role; v != nil {
		where, args = append(where, fmt.Sprintf("member.role = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("member.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	query := `
	SELECT
		principal.id AS user_id,
		principal.email,
		principal.name,
		principal.type,
		principal.password_hash,
		principal.mfa_config,
		principal.phone,
		member.role,
		member.row_status AS row_status
	FROM principal
	LEFT JOIN member ON principal.id = member.principal_id
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
		var role, rowStatus sql.NullString
		var mfaConfigBytes []byte
		if err := rows.Scan(
			&userMessage.ID,
			&userMessage.Email,
			&userMessage.Name,
			&userMessage.Type,
			&userMessage.PasswordHash,
			&mfaConfigBytes,
			&userMessage.Phone,
			&role,
			&rowStatus,
		); err != nil {
			return nil, err
		}
		if role.Valid {
			userMessage.Role = api.Role(role.String)
		} else if userMessage.ID == api.SystemBotID {
			userMessage.Role = api.Owner
		} else {
			userMessage.Role = api.Developer
		}
		if rowStatus.Valid {
			userMessage.MemberDeleted = convertRowStatusToDeleted(rowStatus.String)
		} else if userMessage.ID != api.SystemBotID {
			userMessage.MemberDeleted = true
		}
		mfaConfig := storepb.MFAConfig{}
		decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := decoder.Unmarshal(mfaConfigBytes, &mfaConfig); err != nil {
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
	role := api.Developer
	firstMember := count == 0
	// Grant the member Owner role if there is no existing member.
	if firstMember {
		role = api.Owner
	} else if create.Role != "" {
		role = create.Role
	}

	if _, err := tx.ExecContext(ctx, `
			INSERT INTO member (
				creator_id,
				updater_id,
				status,
				role,
				principal_id
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
		creatorID,
		creatorID,
		api.Active,
		role,
		userID,
	); err != nil {
		return nil, err
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
		Role:         role,
	}
	s.userIDCache.Store(user.ID, user)
	return user, nil
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, userID int, patch *UpdateUserMessage, updaterID int) (*UserMessage, error) {
	if userID == api.SystemBotID {
		return nil, errors.Errorf("cannot update system bot")
	}

	principalSet, principalArgs := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", updaterID)}
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
	principalArgs = append(principalArgs, userID)

	memberSet, memberArgs := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", updaterID)}
	if v := patch.Role; v != nil {
		memberSet, memberArgs = append(memberSet, fmt.Sprintf("role = $%d", len(memberArgs)+1)), append(memberArgs, *v)
	}
	if v := patch.Delete; v != nil {
		rowStatus := api.Normal
		if *patch.Delete {
			rowStatus = api.Archived
		}
		memberSet, memberArgs = append(memberSet, fmt.Sprintf(`"row_status" = $%d`, len(memberArgs)+1)), append(memberArgs, rowStatus)
	}
	memberArgs = append(memberArgs, userID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	user := &UserMessage{}
	var mfaConfigBytes []byte
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE principal
		SET `+strings.Join(principalSet, ", ")+`
		WHERE id = $%d
		RETURNING id, email, name, type, password_hash, mfa_config, phone
	`, len(principalArgs)),
		principalArgs...,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Type,
		&user.PasswordHash,
		&mfaConfigBytes,
		&user.Phone,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	mfaConfig := storepb.MFAConfig{}
	decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := decoder.Unmarshal(mfaConfigBytes, &mfaConfig); err != nil {
		return nil, err
	}
	user.MFAConfig = &mfaConfig

	var rowStatus string
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE member
			SET `+strings.Join(memberSet, ", ")+`
			WHERE principal_id = $%d
			RETURNING role, row_status
		`, len(memberArgs)),
		memberArgs...,
	).Scan(
		&user.Role,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	user.MemberDeleted = convertRowStatusToDeleted(rowStatus)

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.userIDCache.Store(user.ID, user)
	if patch.Email != nil && patch.Phone != nil {
		s.projectIDPolicyCache = sync.Map{}
		s.projectPolicyCache = sync.Map{}
	}
	return user, nil
}
