package store

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

var systemBotUser = &UserMessage{
	ID:    common.SystemBotID,
	Name:  "Bytebase",
	Email: "support@bytebase.com",
	Type:  storepb.PrincipalType_SYSTEM_BOT,
}

// FindUserMessage is the message for finding users.
type FindUserMessage struct {
	ID          *int
	Email       *string
	ShowDeleted bool
	Type        *storepb.PrincipalType
	Limit       *int
	Offset      *int
	Filter      *ListResourceFilter
	ProjectID   *string
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
	Type          storepb.PrincipalType
	PasswordHash  string
	MemberDeleted bool
	MFAConfig     *storepb.MFAConfig
	Profile       *storepb.UserProfile
	// Phone conforms E.164 format.
	Phone string
	// output only
	CreatedAt time.Time
	// The group email list
	Groups []string
}

type UserStat struct {
	Type    storepb.PrincipalType
	Deleted bool
	Count   int
}

// GetSystemBotUser gets the system bot.
func (s *Store) GetSystemBotUser(ctx context.Context) *UserMessage {
	user, err := s.GetUserByID(ctx, common.SystemBotID)
	if err != nil {
		slog.Error("failed to find system bot", slog.Int("id", common.SystemBotID), log.BBError(err))
		return systemBotUser
	}
	if user == nil {
		return systemBotUser
	}
	return user
}

// GetUserByID gets the user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int) (*UserMessage, error) {
	if v, ok := s.userIDCache.Get(id); ok && s.enableCache {
		return v, nil
	}

	if err := s.listAndCacheAllUsers(ctx); err != nil {
		return nil, err
	}

	user, _ := s.userIDCache.Get(id)
	return user, nil
}

// GetUserByEmail gets the user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*UserMessage, error) {
	if v, ok := s.userEmailCache.Get(email); ok && s.enableCache {
		return v, nil
	}

	if err := s.listAndCacheAllUsers(ctx); err != nil {
		return nil, err
	}

	user, _ := s.userEmailCache.Get(email)
	return user, nil
}

func (s *Store) StatUsers(ctx context.Context) ([]*UserStat, error) {
	q := qb.Q().Space(`
		SELECT
			COUNT(*),
			type,
			deleted
		FROM principal
		GROUP BY type, deleted
	`)
	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*UserStat

	for rows.Next() {
		var stat UserStat
		var typeString string
		if err := rows.Scan(
			&stat.Count,
			&typeString,
			&stat.Deleted,
		); err != nil {
			return nil, err
		}
		if typeValue, ok := storepb.PrincipalType_value[typeString]; ok {
			stat.Type = storepb.PrincipalType(typeValue)
		} else {
			return nil, errors.Errorf("invalid principal type string: %s", typeString)
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
	tx, err := s.GetDB().BeginTx(ctx, nil)
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
func (s *Store) listAndCacheAllUsers(ctx context.Context) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	users, err := listUserImpl(ctx, tx, &FindUserMessage{ShowDeleted: true})
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	for _, user := range users {
		s.userIDCache.Add(user.ID, user)
		s.userEmailCache.Add(user.Email, user)
	}
	return nil
}

func listUserImpl(ctx context.Context, txn *sql.Tx, find *FindUserMessage) ([]*UserMessage, error) {
	var with, join string
	if v := find.ProjectID; v != nil {
		with = `WITH all_members AS (
			SELECT
				jsonb_array_elements_text(jsonb_array_elements(policy.payload->'bindings')->'members') AS member,
				jsonb_array_elements(policy.payload->'bindings')->>'role' AS role
			FROM policy
			WHERE ((resource_type = '` + storepb.Policy_PROJECT.String() + `' AND resource = 'projects/` + *v + `') OR resource_type = '` + storepb.Policy_WORKSPACE.String() + `') AND type = '` + storepb.Policy_IAM.String() + `'
		),
		project_members AS (
			SELECT ARRAY_AGG(member) AS members FROM all_members WHERE role NOT LIKE 'roles/workspace%'
		)`
		join = `INNER JOIN project_members ON (CONCAT('users/', principal.id) = ANY(project_members.members) OR '` + common.AllUsers + `' = ANY(project_members.members))`
	}

	// Join the user_group table to find groups for each user.
	// The user will be stored in the user_group.payload.members.member field, the member is in the "users/{id}" format
	if strings.HasPrefix(with, "WITH") {
		with += ","
	} else {
		with = "WITH"
	}

	q := qb.Q().Space(with + ` user_groups AS (
		SELECT
			principal.id AS user_id,
			COALESCE(ARRAY_AGG(user_group.email ORDER BY user_group.email) FILTER (WHERE user_group.email IS NOT NULL), '{}') AS groups
		FROM principal
		LEFT JOIN user_group ON EXISTS (
			SELECT 1 FROM jsonb_array_elements(user_group.payload->'members') AS m
			WHERE m->>'member' = CONCAT('users/', principal.id)
		)
		GROUP BY principal.id
	)
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
		principal.created_at,
		user_groups.groups
	FROM principal
	INNER JOIN user_groups ON principal.id = user_groups.user_id
	` + join + ` WHERE TRUE`)

	if filter := find.Filter; filter != nil {
		// Convert $1, $2, etc. to ? for qb
		q.And(ConvertDollarPlaceholders(filter.Where), filter.Args...)
	}
	if v := find.ID; v != nil {
		q.And("principal.id = ?", *v)
	}
	if v := find.Email; v != nil {
		if *v == common.AllUsers {
			q.And("principal.email = ?", *v)
		} else {
			q.And("principal.email = ?", strings.ToLower(*v))
		}
	}
	if v := find.Type; v != nil {
		q.And("principal.type = ?", v.String())
	}
	if !find.ShowDeleted {
		q.And("principal.deleted = ?", false)
	}

	q.Space("ORDER BY type DESC, created_at ASC")

	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var userMessages []*UserMessage
	rows, err := txn.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userMessage UserMessage
		var mfaConfigBytes []byte
		var profileBytes []byte
		var typeString string
		var groups pq.StringArray
		if err := rows.Scan(
			&userMessage.ID,
			&userMessage.MemberDeleted,
			&userMessage.Email,
			&userMessage.Name,
			&typeString,
			&userMessage.PasswordHash,
			&mfaConfigBytes,
			&userMessage.Phone,
			&profileBytes,
			&userMessage.CreatedAt,
			&groups,
		); err != nil {
			return nil, err
		}
		userMessage.Groups = []string(groups)
		if typeValue, ok := storepb.PrincipalType_value[typeString]; ok {
			userMessage.Type = storepb.PrincipalType(typeValue)
		} else {
			return nil, errors.Errorf("invalid principal type string: %s", typeString)
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

	tx, err := s.GetDB().BeginTx(ctx, nil)
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

	q := qb.Q().Space(`
		INSERT INTO principal (
			email,
			name,
			type,
			password_hash,
			phone,
			profile
		)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, created_at
	`, create.Email, create.Name, create.Type.String(), create.PasswordHash, create.Phone, profileBytes)

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var userID int
	if err := tx.QueryRowContext(ctx, sql, args...).Scan(&userID, &create.CreatedAt); err != nil {
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
	if currentUser.ID == common.SystemBotID {
		return nil, errors.Errorf("cannot update system bot")
	}

	q := qb.Q().Space("UPDATE principal SET")
	hasUpdate := false

	if v := patch.Delete; v != nil {
		if hasUpdate {
			q.Join(", ", "deleted = ?", *v)
		} else {
			q.Space("deleted = ?", *v)
			hasUpdate = true
		}
	}
	if v := patch.Email; v != nil {
		if hasUpdate {
			q.Join(", ", "email = ?", strings.ToLower(*v))
		} else {
			q.Space("email = ?", strings.ToLower(*v))
			hasUpdate = true
		}
	}
	if v := patch.Name; v != nil {
		if hasUpdate {
			q.Join(", ", "name = ?", *v)
		} else {
			q.Space("name = ?", *v)
			hasUpdate = true
		}
	}
	if v := patch.PasswordHash; v != nil {
		if hasUpdate {
			q.Join(", ", "password_hash = ?", *v)
		} else {
			q.Space("password_hash = ?", *v)
			hasUpdate = true
		}
		if patch.Profile == nil {
			patch.Profile = currentUser.Profile
			patch.Profile.LastChangePasswordTime = timestamppb.New(time.Now())
		}
	}
	if v := patch.Phone; v != nil {
		if hasUpdate {
			q.Join(", ", "phone = ?", *v)
		} else {
			q.Space("phone = ?", *v)
			hasUpdate = true
		}
	}
	if v := patch.MFAConfig; v != nil {
		mfaConfigBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		if hasUpdate {
			q.Join(", ", "mfa_config = ?", mfaConfigBytes)
		} else {
			q.Space("mfa_config = ?", mfaConfigBytes)
			hasUpdate = true
		}
	}
	if v := patch.Profile; v != nil {
		profileBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		if hasUpdate {
			q.Join(", ", "profile = ?", profileBytes)
		} else {
			q.Space("profile = ?", profileBytes)
			hasUpdate = true
		}
	}

	if !hasUpdate {
		return currentUser, nil
	}

	q.Where("id = ?", currentUser.ID)

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
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
