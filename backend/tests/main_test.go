package tests

import (
	"context"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	webhookplugin "github.com/bytebase/bytebase/backend/plugin/webhook"
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	code, err := startMain(ctx, m)
	if err != nil {
		panic(err)
	}

	if code != 0 {
		panic("tests failed")
	}
}

func startMain(ctx context.Context, m *testing.M) (int, error) {
	pgContainer, err := getPgContainer(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if pgContainer != nil {
			pgContainer.Close(ctx)
		}
	}()
	externalPgHost = pgContainer.host
	externalPgPort = pgContainer.port

	// Seeded once here, before any test starts, and never mutated afterwards:
	// ValidateWebhookURL reads this map without synchronization, and the tests
	// that add webhooks run in parallel. A test that set and deleted its own
	// entry would be a concurrent map write against those readers, which is
	// fatal to the process rather than merely racy.
	for _, webhookType := range []storepb.WebhookType{storepb.WebhookType_SLACK, storepb.WebhookType_DINGTALK} {
		webhookplugin.TestOnlyAllowedDomains[webhookType] = []string{"127.0.0.1", "localhost", "[::1]"}
	}

	return m.Run(), nil
}
