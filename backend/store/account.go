package store

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

type AccountStat struct {
	Type    v1pb.ActuatorInfo_AccountStat_Type
	Deleted bool
	Count   int
}

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
			ID:            sa.ID,
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
			ID:            wi.ID,
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

func (s *Store) StatAccounts(ctx context.Context) ([]*AccountStat, error) {
	var stats []*AccountStat

	tableTypes := []struct {
		table       string
		accountType v1pb.ActuatorInfo_AccountStat_Type
	}{
		{"principal", v1pb.ActuatorInfo_AccountStat_USER},
		{"service_account", v1pb.ActuatorInfo_AccountStat_SERVICE_ACCOUNT},
		{"workload_identity", v1pb.ActuatorInfo_AccountStat_WORKLOAD_IDENTITY},
	}

	for _, tt := range tableTypes {
		if err := func() error {
			rows, err := s.GetDB().QueryContext(ctx, "SELECT COUNT(*), deleted FROM "+tt.table+" GROUP BY deleted")
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var stat AccountStat
				if err := rows.Scan(&stat.Count, &stat.Deleted); err != nil {
					return err
				}
				stat.Type = tt.accountType
				stats = append(stats, &stat)
			}
			if err := rows.Err(); err != nil {
				return errors.Wrapf(err, "failed to scan rows")
			}
			return nil
		}(); err != nil {
			return nil, err
		}
	}

	return stats, nil
}
