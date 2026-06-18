package server

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestMarshalSQLReviewConfigSnapshot(t *testing.T) {
	response := &v1pb.ListReviewConfigsResponse{
		ReviewConfigs: []*v1pb.ReviewConfig{
			{
				Name:      "reviewConfigs/basic",
				Title:     "Basic",
				Enabled:   true,
				Resources: []string{"environments/prod"},
				Rules: []*v1pb.SQLReviewRule{
					{
						Type:   v1pb.SQLReviewRule_NAMING_TABLE,
						Level:  v1pb.SQLReviewRule_ERROR,
						Engine: v1pb.Engine_MYSQL,
						Payload: &v1pb.SQLReviewRule_NamingPayload{
							NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
								MaxLength: 64,
								Format:    "^[a-z]+$",
							},
						},
					},
				},
			},
		},
	}

	snapshot, err := marshalSQLReviewConfigSnapshot(response)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snapshot, `"reviewConfigs"`) {
		t.Fatalf("snapshot %q does not contain reviewConfigs", snapshot)
	}
	if !strings.Contains(snapshot, `"namingPayload"`) {
		t.Fatalf("snapshot %q does not contain rule payload", snapshot)
	}

	got := &v1pb.ListReviewConfigsResponse{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(snapshot), got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(response, got) {
		t.Fatalf("roundtrip mismatch\n got: %v\nwant: %v", got, response)
	}
}

func TestBuildSQLReviewTelemetryIdentity(t *testing.T) {
	email, domains := buildSQLReviewTelemetryIdentity([]*store.UserMessage{
		{ID: 30, Email: "carol@example.com"},
		{ID: 10, Email: "alice@test.com"},
		{ID: 20, Email: "bob@example.com"},
		{ID: 40, Email: "invalid-email"},
	})

	if email != "alice@test.com" {
		t.Fatalf("email = %q, want %q", email, "alice@test.com")
	}
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "test.com" {
		t.Fatalf("domains = %v, want %v", domains, []string{"example.com", "test.com"})
	}
}
