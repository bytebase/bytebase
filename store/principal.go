package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// GetPrincipalList gets a list of Principal instances.
func (s *Store) GetPrincipalList(ctx context.Context) ([]*api.Principal, error) {
	users, err := s.ListUsers(ctx, &FindUserMessage{ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find Principal list")
	}
	var principalList []*api.Principal
	for _, user := range users {
		principal, err := composePrincipal(user)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Principal role with user[%+v]", user)
		}
		if principal != nil {
			principalList = append(principalList, principal)
		}
	}
	return principalList, nil
}

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
}

// UpdateUserMessage is the message to update a user.
type UpdateUserMessage struct {
	Email        *string
	Name         *string
	PasswordHash *string
	Role         *api.Role
	Delete       *bool
}

// UserMessage is the message for an user.
type UserMessage struct {
	ID                         int
	Email                      string
	Name                       string
	Type                       api.PrincipalType
	PasswordHash               string
	IdentityProviderResourceID *string
	// IdentityProviderUserInfo is a raw json string from userinfo api response.
	// e.g. for GitHub's authenticated user api, it looks like the following structure..
	// https://docs.github.com/en/rest/users/users?apiVersion=2022-11-28#get-the-authenticated-user
	IdentityProviderUserInfo string
	Role                     api.Role
	MemberDeleted            bool
}

// ListUsers list all users.
func (s *Store) ListUsers(ctx context.Context, find *FindUserMessage) ([]*UserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	users, err := s.listUserImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	for _, user := range users {
		s.userIDCache.Store(user.ID, user)
		s.userEmailCache.Store(user.Email, user)
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
		return nil, FormatError(err)
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
		return nil, FormatError(err)
	}

	s.userIDCache.Store(user.ID, user)
	s.userEmailCache.Store(user.Email, user)
	return user, nil
}

// GetUserByEmail gets the user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*UserMessage, error) {
	// We use lower-case for emails.
	email = strings.ToLower(email)

	if user, ok := s.userEmailCache.Load(email); ok {
		return user.(*UserMessage), nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	users, err := s.listUserImpl(ctx, tx, &FindUserMessage{Email: &email, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	if len(users) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d users with email %q, expect 1", len(users), email)}
	}
	user := users[0]
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	s.userIDCache.Store(user.ID, user)
	s.userEmailCache.Store(user.Email, user)
	return user, nil
}

// GetUserByEmailV2 gets an instance of Principal.
func (*Store) listUserImpl(ctx context.Context, tx *Tx, find *FindUserMessage) ([]*UserMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("principal.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Email; v != nil {
		where, args = append(where, fmt.Sprintf("principal.email = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Role; v != nil {
		where, args = append(where, fmt.Sprintf("member.role = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("member.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var userMessages []*UserMessage
	if common.FeatureFlag(common.FeatureFlagNoop) {
		rows, err := tx.QueryContext(ctx, `
			SELECT
				principal.id AS user_id,
				principal.email,
				principal.name,
				principal.type,
				principal.password_hash,
				principal.idp_user_info,
				member.role,
				member.row_status AS row_status,
				idp.resource_id AS idp_resource_id
			FROM principal
			LEFT JOIN member ON principal.id = member.principal_id
			LEFT JOIN idp ON principal.idp_id = idp.id
			WHERE `+strings.Join(where, " AND "),
			args...,
		)
		if err != nil {
			return nil, FormatError(err)
		}
		defer rows.Close()
		for rows.Next() {
			var userMessage UserMessage
			var role, rowStatus, idpResourceID sql.NullString
			if err := rows.Scan(
				&userMessage.ID,
				&userMessage.Email,
				&userMessage.Name,
				&userMessage.Type,
				&userMessage.PasswordHash,
				&userMessage.IdentityProviderUserInfo,
				&role,
				&rowStatus,
				&idpResourceID,
			); err != nil {
				return nil, FormatError(err)
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
			if idpResourceID.Valid {
				userMessage.IdentityProviderResourceID = &idpResourceID.String
			}
			userMessages = append(userMessages, &userMessage)
		}
	} else {
		rows, err := tx.QueryContext(ctx, `
			SELECT
				principal.id AS user_id,
				principal.email,
				principal.name,
				principal.type,
				principal.password_hash,
				member.role,
				member.row_status AS row_status
			FROM principal
			LEFT JOIN member ON principal.id = member.principal_id
			WHERE `+strings.Join(where, " AND "),
			args...,
		)
		if err != nil {
			return nil, FormatError(err)
		}
		defer rows.Close()
		for rows.Next() {
			var userMessage UserMessage
			var role, rowStatus sql.NullString
			if err := rows.Scan(
				&userMessage.ID,
				&userMessage.Email,
				&userMessage.Name,
				&userMessage.Type,
				&userMessage.PasswordHash,
				&role,
				&rowStatus,
			); err != nil {
				return nil, FormatError(err)
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
			userMessages = append(userMessages, &userMessage)
		}
	}
	return userMessages, nil
}

// CreateUser creates an user.
func (s *Store) CreateUser(ctx context.Context, create *UserMessage, creatorID int) (*UserMessage, error) {
	// We use lower-case for emails.
	create.Email = strings.ToLower(create.Email)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM principal WHERE type = $1`,
		api.EndUser,
	).Scan(&count); err != nil {
		return nil, err
	}
	role := api.Developer
	// Grant the member Owner role if there is no existing Owner member.
	if count == 0 {
		role = api.Owner
	}

	set := []string{"creator_id", "updater_id", "email", "name", "type", "password_hash"}
	args := []interface{}{creatorID, creatorID, create.Email, create.Name, create.Type, create.PasswordHash}
	// Set idp fields into principal only when the related id is not null.
	if common.FeatureFlag(common.FeatureFlagNoop) {
		if create.IdentityProviderResourceID != nil {
			identityProvider, err := s.GetIdentityProvider(ctx, &FindIdentityProviderMessage{
				ResourceID: create.IdentityProviderResourceID,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find identity provider with resource id %s", create.IdentityProviderResourceID)
			}
			set, args = append(set, "idp_id"), append(args, identityProvider.UID)
			set, args = append(set, "idp_user_info"), append(args, create.IdentityProviderUserInfo)
		}
	}
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
		return nil, FormatError(err)
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
		return nil, FormatError(err)
	}

	user := &UserMessage{
		ID:           userID,
		Email:        create.Email,
		Name:         create.Name,
		Type:         create.Type,
		PasswordHash: create.PasswordHash,
		Role:         role,
	}
	if common.FeatureFlag(common.FeatureFlagNoop) {
		user.IdentityProviderResourceID = create.IdentityProviderResourceID
		user.IdentityProviderUserInfo = create.IdentityProviderUserInfo
	}
	s.userIDCache.Store(user.ID, user)
	s.userEmailCache.Store(user.Email, user)
	return user, nil
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, userID int, patch *UpdateUserMessage, updaterID int) (*UserMessage, error) {
	if userID == api.SystemBotID {
		return nil, errors.Errorf("cannot update system bot")
	}
	oldUser, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	principalSet, principalArgs := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", updaterID)}
	if v := patch.Email; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("email = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.Name; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("name = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.PasswordHash; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("password_hash = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	principalArgs = append(principalArgs, userID)

	memberSet, memberArgs := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", updaterID)}
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
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	user := &UserMessage{}
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE principal
			SET `+strings.Join(principalSet, ", ")+`
			WHERE id = $%d
			RETURNING id, email, name, type, password_hash
		`, len(principalArgs)),
		principalArgs...,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Type,
		&user.PasswordHash,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}
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
		return nil, FormatError(err)
	}
	user.MemberDeleted = convertRowStatusToDeleted(rowStatus)

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	s.userEmailCache.Delete(oldUser.Email)
	s.userIDCache.Store(user.ID, user)
	s.userEmailCache.Store(user.Email, user)
	return user, nil
}
