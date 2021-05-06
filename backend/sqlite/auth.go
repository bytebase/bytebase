package sqlite

import "github.com/bytebase/bytebase/api"

var (
	_ api.AuthService = (*AuthService)(nil)
)

// AuthService represents a service for managing auth.
type AuthService struct {
	db *DB
}

// NewAuthService returns a new instance of AuthService.
func NewAuthService(db *DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) FindPrincipalByEmail(email string) (*api.Principal, error) {
	return &api.Principal{ID: 1, Email: "foo@example.com"}, nil
}
