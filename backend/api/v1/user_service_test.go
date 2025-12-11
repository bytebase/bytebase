package v1

import (
	"context"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestConvertToUser(t *testing.T) {
	tests := []struct {
		name        string
		user        *store.UserMessage
		wantName    string
		wantEmail   string
		description string
	}{
		{
			name: "user with email",
			user: &store.UserMessage{
				ID:    1,
				Email: "test@example.com",
				Name:  "Test User",
				Type:  storepb.PrincipalType_END_USER,
				Profile: &storepb.UserProfile{
					Source: "BYTEBASE",
				},
			},
			wantName:    "users/test@example.com",
			wantEmail:   "test@example.com",
			description: "should format name using email when email is provided",
		},
		{
			name: "user with empty email",
			user: &store.UserMessage{
				ID:    42,
				Email: "",
				Name:  "Test User",
				Type:  storepb.PrincipalType_END_USER,
				Profile: &storepb.UserProfile{
					Source: "BYTEBASE",
				},
			},
			wantName:    "users/42",
			wantEmail:   "",
			description: "should format name using UID when email is empty",
		},
		{
			name: "service account with email",
			user: &store.UserMessage{
				ID:    2,
				Email: "service@example.com",
				Name:  "Service Account",
				Type:  storepb.PrincipalType_SERVICE_ACCOUNT,
				Profile: &storepb.UserProfile{
					Source: "BYTEBASE",
				},
			},
			wantName:    "users/service@example.com",
			wantEmail:   "service@example.com",
			description: "should format service account name using email",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToUser(ctx, tt.user)
			if got.Name != tt.wantName {
				t.Errorf("convertToUser() name = %v, want %v. %s", got.Name, tt.wantName, tt.description)
			}
			if got.Email != tt.wantEmail {
				t.Errorf("convertToUser() email = %v, want %v. %s", got.Email, tt.wantEmail, tt.description)
			}
		})
	}
}
