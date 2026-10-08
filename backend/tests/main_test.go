package tests

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	webhookplugin "github.com/bytebase/bytebase/backend/plugin/webhook"
)

func TestMain(m *testing.M) {
	code, err := startMain(m)
	if err != nil {
		panic(err)
	}

	if code != 0 {
		panic("tests failed")
	}
}

func startMain(m *testing.M) (int, error) {
	// The package's own Postgres: every server the tests start takes a metadata
	// database on it, as does every test that builds a Store without a server.
	pgContainer, err := testcontainer.StartSharedPg()
	if err != nil {
		return 0, err
	}
	defer testcontainer.CloseShared()
	// Runs before CloseShared takes the Postgres out from under it.
	defer func() {
		if sharedServerCtl != nil {
			_ = sharedServerCtl.Close(context.Background())
		}
	}()
	externalPgHost = pgContainer.GetHost()
	externalPgPort = pgContainer.GetPort()

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
