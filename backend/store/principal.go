package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	if user.IdentityProviderResourceID != nil {
		principal.IdentityProviderName = *user.IdentityProviderResourceID
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
	// IdentityProviderResourceID is the name of the identity provider related with the user.
	// If set with empty string, then only those users that are not from the idp will be found.
	IdentityProviderResourceID *string
	// Available only if the IdentityProviderResourceID is not nil.
	IdentityProviderUserIdentifier string
}

// UpdateUserMessage is the message to update a user.
type UpdateUserMessage struct {
	Email                    *string
	Name                     *string
	PasswordHash             *string
	Role                     *api.Role
	IdentityProviderUserInfo *storepb.IdentityProviderUserInfo
	Delete                   *bool
}

// UserMessage is the message for an user.
type UserMessage struct {
	ID int
	// Email must be lower case.
	Email                      string
	Name                       string
	Type                       api.PrincipalType
	PasswordHash               string
	IdentityProviderResourceID *string
	IdentityProviderUserInfo   *storepb.IdentityProviderUserInfo
	Role                       api.Role
	MemberDeleted              bool
}

// GetUser gets an user.
func (s *Store) GetUser(ctx context.Context, find *FindUserMessage) (*UserMessage, error) {
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
	return user, nil
}

func (s *Store) listUserImpl(ctx context.Context, tx *Tx, find *FindUserMessage) ([]*UserMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	// Do not to select those archived IdP user.
	where, args = append(where, fmt.Sprintf("(principal.idp_id IS NULL OR idp.row_status = $%d)", len(args)+1)), append(args, api.Normal)
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
	if find.IdentityProviderResourceID != nil {
		if *find.IdentityProviderResourceID == "" {
			where = append(where, "principal.idp_id IS NULL")
		} else {
			// Get identity provider's UID with resource id.
			identityProvider, err := s.GetIdentityProvider(ctx, &FindIdentityProviderMessage{
				ResourceID: find.IdentityProviderResourceID,
			})
			if err != nil {
				return nil, err
			}
			where, args = append(where, fmt.Sprintf("principal.idp_id = $%d", len(args)+1)), append(args, identityProvider.UID)
			where = append(where, fmt.Sprintf("principal.idp_user_info->>'identifier' = '%s'", find.IdentityProviderUserIdentifier))
		}
	}

	var userMessages []*UserMessage
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
		var idpUserInfo string
		if err := rows.Scan(
			&userMessage.ID,
			&userMessage.Email,
			&userMessage.Name,
			&userMessage.Type,
			&userMessage.PasswordHash,
			&idpUserInfo,
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
			var identityProviderUserInfo storepb.IdentityProviderUserInfo
			if err := json.Unmarshal([]byte(idpUserInfo), &identityProviderUserInfo); err != nil {
				return nil, err
			}
			userMessage.IdentityProviderUserInfo = &identityProviderUserInfo
		}
		userMessages = append(userMessages, &userMessage)
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

	set := []string{"creator_id", "updater_id", "email", "name", "type", "password_hash"}
	args := []interface{}{creatorID, creatorID, create.Email, create.Name, create.Type, create.PasswordHash}
	// Set idp fields into principal only when the related id is not null.
	if create.IdentityProviderResourceID != nil {
		identityProvider, err := s.GetIdentityProvider(ctx, &FindIdentityProviderMessage{
			ResourceID: create.IdentityProviderResourceID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find identity provider with resource id %s", *create.IdentityProviderResourceID)
		}
		set, args = append(set, "idp_id"), append(args, identityProvider.UID)
		userInfoBytes, err := protojson.Marshal(create.IdentityProviderUserInfo)
		if err != nil {
			return nil, err
		}
		set, args = append(set, "idp_user_info"), append(args, string(userInfoBytes))
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
		ID:                         userID,
		Email:                      create.Email,
		Name:                       create.Name,
		Type:                       create.Type,
		PasswordHash:               create.PasswordHash,
		Role:                       role,
		IdentityProviderResourceID: create.IdentityProviderResourceID,
		IdentityProviderUserInfo:   create.IdentityProviderUserInfo,
	}
	s.userIDCache.Store(user.ID, user)
	return user, nil
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, userID int, patch *UpdateUserMessage, updaterID int) (*UserMessage, error) {
	if userID == api.SystemBotID {
		return nil, errors.Errorf("cannot update system bot")
	}

	principalSet, principalArgs := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", updaterID)}
	if v := patch.Email; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("email = $%d", len(principalArgs)+1)), append(principalArgs, strings.ToLower(*v))
	}
	if v := patch.Name; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("name = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.PasswordHash; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("password_hash = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.IdentityProviderUserInfo; v != nil {
		userInfoBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("idp_user_info = $%d", len(principalArgs)+1)), append(principalArgs, string(userInfoBytes))
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
	var idpID sql.NullInt32
	var idpUserInfo string
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE principal
			SET `+strings.Join(principalSet, ", ")+`
			WHERE id = $%d
			RETURNING id, email, name, type, password_hash, idp_id, idp_user_info
		`, len(principalArgs)),
		principalArgs...,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Type,
		&user.PasswordHash,
		&idpID,
		&idpUserInfo,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}
	if idpID.Valid {
		value := int(idpID.Int32)
		idp, err := s.GetIdentityProvider(ctx, &FindIdentityProviderMessage{
			UID: &value,
		})
		if err != nil {
			return nil, FormatError(err)
		}
		user.IdentityProviderResourceID = &idp.ResourceID
		var identityProviderUserInfo storepb.IdentityProviderUserInfo
		if err := json.Unmarshal([]byte(idpUserInfo), &identityProviderUserInfo); err != nil {
			return nil, err
		}
		user.IdentityProviderUserInfo = &identityProviderUserInfo
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
	s.userIDCache.Store(user.ID, user)
	return user, nil
}
