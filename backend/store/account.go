package store

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// GetPrincipalByEmail gets any principal (user, service account, or workload identity) by email.
// This is used by the auth layer where the JWT subject can be any principal type.
// It determines the principal type by the email format and calls the appropriate method.
func (s *Store) GetPrincipalByEmail(ctx context.Context, email string) (*UserMessage, error) {
	// Use the unified cache first for all types
	if v, ok := s.userEmailCache.Get(email); ok && s.enableCache {
		return v, nil
	}

	// Determine principal type by email format and query accordingly
	if common.IsServiceAccountEmail(email) {
		sa, err := s.GetServiceAccountByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if sa == nil {
			return nil, nil
		}
		// Convert to UserMessage for compatibility with auth layer
		user := &UserMessage{
			Email:         sa.Email,
			Name:          sa.Name,
			PasswordHash:  sa.ServiceKeyHash,
			Type:          storepb.PrincipalType_SERVICE_ACCOUNT,
			MemberDeleted: sa.MemberDeleted,
			MFAConfig:     &storepb.MFAConfig{},
			Profile:       &storepb.UserProfile{},
		}
		s.userEmailCache.Add(user.Email, user)
		return user, nil
	}

	if common.IsWorkloadIdentityEmail(email) {
		wi, err := s.GetWorkloadIdentityByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if wi == nil {
			return nil, nil
		}
		// Convert to UserMessage for compatibility with auth layer
		user := &UserMessage{
			Email:         wi.Email,
			Name:          wi.Name,
			Type:          storepb.PrincipalType_WORKLOAD_IDENTITY,
			MemberDeleted: wi.MemberDeleted,
			MFAConfig:     &storepb.MFAConfig{},
			Profile:       &storepb.UserProfile{},
		}
		s.userEmailCache.Add(user.Email, user)
		return user, nil
	}

	// Default to end user lookup.
	return s.GetUserByEmail(ctx, email)
}
