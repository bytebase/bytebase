package store

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

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
}

// FindUserMessage is the message for finding users.
type FindUserMessage struct {
	ID          *int
	Email       *string
	ShowDeleted bool
	Type        *api.PrincipalType
	Limit       *int
	Offset      *int
	Filter      *ListResourceFilter
}

// UpdateUserMessage is the message to update a user.
type UpdateUserMessage struct {
	Email        *string
	Name         *string
	PasswordHash *string
	Delete       *bool
	MFAConfig    *storepb.MFAConfig
	Profile      *storepb.UserProfile
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
	MemberDeleted bool
	MFAConfig     *storepb.MFAConfig
	Profile       *storepb.UserProfile
	// Phone conforms E.164 format.
	Phone string
	// output only
	CreatedAt time.Time
}

type UserStat struct {
	Type    api.PrincipalType
	Deleted bool
	Count   int
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

	if _, err := s.listAndCacheAllUsers(ctx); err != nil {
		return nil, err
	}

	user, _ := s.userIDCache.Get(id)
	return user, nil
}

// GetUserByEmail gets the user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*UserMessage, error) {
	if v, ok := s.userEmailCache.Get(email); ok {
		return v, nil
	}

	if _, err := s.listAndCacheAllUsers(ctx); err != nil {
		return nil, err
	}

	user, _ := s.userEmailCache.Get(email)
	return user, nil
}

func (s *Store) StatUsers(ctx context.Context) ([]*UserStat, error) {
	rows, err := s.db.db.QueryContext(ctx, `
	SELECT
		COUNT(*),
		type,
		deleted
	FROM principal
	GROUP BY type, deleted`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*UserStat

	for rows.Next() {
		var stat UserStat
		if err := rows.Scan(
			&stat.Count,
			&stat.Type,
			&stat.Deleted,
		); err != nil {
			return nil, err
		}
		stats = append(stats, &stat)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan rows")
	}

	return stats, nil
}

// ListUsers list users.
func (s *Store) ListUsers(ctx context.Context, find *FindUserMessage) ([]*UserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	users, err := listUserImpl(ctx, tx, find)
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

	users, err := listUserImpl(ctx, tx, &FindUserMessage{ShowDeleted: true})
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

func listUserImpl(ctx context.Context, tx *Tx, find *FindUserMessage) ([]*UserMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if filter := find.Filter; filter != nil {
		where = append(where, filter.Where)
		args = append(args, filter.Args...)
	}
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
		where, args = append(where, fmt.Sprintf("principal.deleted = $%d", len(args)+1)), append(args, false)
	}

	query := `
	SELECT
		principal.id AS user_id,
		principal.deleted,
		principal.email,
		principal.name,
		principal.type,
		principal.password_hash,
		principal.mfa_config,
		principal.phone,
		principal.profile,
		principal.created_at
	FROM principal
	WHERE ` + strings.Join(where, " AND ") + ` ORDER BY type DESC, created_at ASC`

	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	var userMessages []*UserMessage
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userMessage UserMessage
		var mfaConfigBytes []byte
		var profileBytes []byte
		if err := rows.Scan(
			&userMessage.ID,
			&userMessage.MemberDeleted,
			&userMessage.Email,
			&userMessage.Name,
			&userMessage.Type,
			&userMessage.PasswordHash,
			&mfaConfigBytes,
			&userMessage.Phone,
			&profileBytes,
			&userMessage.CreatedAt,
		); err != nil {
			return nil, err
		}

		mfaConfig := storepb.MFAConfig{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(mfaConfigBytes, &mfaConfig); err != nil {
			return nil, err
		}
		userMessage.MFAConfig = &mfaConfig
		profile := storepb.UserProfile{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(profileBytes, &profile); err != nil {
			return nil, err
		}
		userMessage.Profile = &profile

		userMessages = append(userMessages, &userMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userMessages, nil
}

// CreateUser creates an user.
func (s *Store) CreateUser(ctx context.Context, create *UserMessage) (*UserMessage, error) {
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

	if create.Profile == nil {
		create.Profile = &storepb.UserProfile{}
	}
	profileBytes, err := protojson.Marshal(create.Profile)
	if err != nil {
		return nil, err
	}

	set := []string{"email", "name", "type", "password_hash", "phone", "profile"}
	args := []any{create.Email, create.Name, create.Type, create.PasswordHash, create.Phone, profileBytes}
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
			RETURNING id, created_at
		`, strings.Join(set, ","), strings.Join(placeholder, ",")),
		args...,
	).Scan(&userID, &create.CreatedAt); err != nil {
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
		CreatedAt:    create.CreatedAt,
		Profile:      create.Profile,
		MFAConfig:    &storepb.MFAConfig{},
	}
	s.userIDCache.Add(user.ID, user)
	s.userEmailCache.Add(user.Email, user)
	return user, nil
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, currentUser *UserMessage, patch *UpdateUserMessage) (*UserMessage, error) {
	if currentUser.ID == api.SystemBotID {
		return nil, errors.Errorf("cannot update system bot")
	}

	principalSet, principalArgs := []string{}, []any{}
	if v := patch.Delete; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("deleted = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.Email; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("email = $%d", len(principalArgs)+1)), append(principalArgs, strings.ToLower(*v))
	}
	if v := patch.Name; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("name = $%d", len(principalArgs)+1)), append(principalArgs, *v)
	}
	if v := patch.PasswordHash; v != nil {
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("password_hash = $%d", len(principalArgs)+1)), append(principalArgs, *v)
		if patch.Profile == nil {
			patch.Profile = currentUser.Profile
			patch.Profile.LastChangePasswordTime = timestamppb.New(time.Now())
		}
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
	if v := patch.Profile; v != nil {
		profileBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		principalSet, principalArgs = append(principalSet, fmt.Sprintf("profile = $%d", len(principalArgs)+1)), append(principalArgs, profileBytes)
	}
	principalArgs = append(principalArgs, currentUser.ID)

	if len(principalSet) == 0 {
		return currentUser, nil
	}

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
