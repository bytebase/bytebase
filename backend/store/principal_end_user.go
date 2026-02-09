package store

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// GetEndUserByEmail gets an end user by email.
func (s *Store) GetEndUserByEmail(ctx context.Context, email string) (*UserMessage, error) {
	if v, ok := s.userEmailCache.Get(email); ok && s.enableCache {
		if v.Type == storepb.PrincipalType_END_USER {
			return v, nil
		}
		return nil, nil
	}

	users, err := s.ListEndUsers(ctx, &FindUserMessage{Email: &email, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}

// listEndUsers lists end users.
func (s *Store) ListEndUsers(ctx context.Context, find *FindUserMessage) ([]*UserMessage, error) {
	userTypes := []storepb.PrincipalType{
		storepb.PrincipalType_END_USER,
	}
	find.UserTypes = &userTypes
	return s.ListUsers(ctx, find)
}

// CreateEndUser creates an end user.
func (s *Store) CreateEndUser(ctx context.Context, create *UserMessage) (*UserMessage, error) {
	if create.Type != storepb.PrincipalType_END_USER {
		return nil, errors.Errorf("expected END_USER type, got %s", create.Type.String())
	}
	return s.CreateUser(ctx, create)
}

// UpdateEndUser updates an end user.
func (s *Store) UpdateEndUser(ctx context.Context, currentUser *UserMessage, patch *UpdateUserMessage) (*UserMessage, error) {
	if currentUser.Type != storepb.PrincipalType_END_USER {
		return nil, errors.Errorf("expected END_USER type, got %s", currentUser.Type.String())
	}
	return s.UpdateUser(ctx, currentUser, patch)
}

// UpdateEndUserEmail updates an end user's email and all related references.
func (s *Store) UpdateEndUserEmail(ctx context.Context, user *UserMessage, newEmail string) (*UserMessage, error) {
	if user.Type != storepb.PrincipalType_END_USER {
		return nil, errors.Errorf("expected END_USER type, got %s", user.Type.String())
	}
	return s.UpdateUserEmail(ctx, user, newEmail)
}

// CreateEndUserMessage is the message for creating an end user.
type CreateEndUserMessage struct {
	Email        string
	Name         string
	PasswordHash string
	Phone        string
	Profile      *storepb.UserProfile
}

// CreateEndUserFromMessage creates an end user from CreateEndUserMessage.
func (s *Store) CreateEndUserFromMessage(ctx context.Context, create *CreateEndUserMessage) (*UserMessage, error) {
	email := strings.ToLower(create.Email)

	profile := create.Profile
	if profile == nil {
		profile = &storepb.UserProfile{}
	}
	profileBytes, err := protojson.Marshal(profile)
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
	`, email, create.Name, storepb.PrincipalType_END_USER.String(), create.PasswordHash, create.Phone, profileBytes)

	sqlStr, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	user := &UserMessage{
		Email:        email,
		Name:         create.Name,
		Type:         storepb.PrincipalType_END_USER,
		PasswordHash: create.PasswordHash,
		Phone:        create.Phone,
		Profile:      profile,
		MFAConfig:    &storepb.MFAConfig{},
	}

	if err := s.GetDB().QueryRowContext(ctx, sqlStr, args...).Scan(&user.ID, &user.CreatedAt); err != nil {
		return nil, err
	}

	s.userEmailCache.Add(user.Email, user)
	return user, nil
}
