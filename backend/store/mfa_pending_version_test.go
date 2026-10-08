package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestPendingMFAVersionSurvivesNanosecondClocks pins the precision contract
// between the pending version and the predicate that matches it.
//
// UpdateUserMFAConfigIfPending compares the stored value as a timestamptz, and
// that type holds microseconds. The cast rounds a nanosecond tail while the
// bound parameter does not, so an instant whose sub-microsecond remainder
// rounds up used to match nothing and the confirmation reported that its own
// enrollment had been superseded.
//
// It never reproduced on macOS, whose clock is microsecond-granular: every
// value it produces is already exact. On Linux, which is nanosecond-granular,
// it struck roughly half of all enrollments — so the instants here are chosen
// rather than sampled from time.Now().
func TestPendingMFAVersionSurvivesNanosecondClocks(t *testing.T) {
	t.Parallel()
	ctx, _, s := newSettingAtomicFixture(t)
	a := require.New(t)

	user, err := s.CreateUser(ctx, &store.UserMessage{
		Email: "enroller@example.com",
		Name:  "enroller",
		Type:  storepb.PrincipalType_END_USER,
	})
	a.NoError(err)

	base := time.Date(2026, 8, 27, 5, 10, 43, 0, time.UTC)
	for _, nanos := range []int{0, 1, 499, 500, 501, 999, 123456000, 123456789, 999999999} {
		// Handed in with the nanosecond tail a Linux clock produces, not
		// pre-rounded: the store owns this contract, so it has to hold for a
		// caller that passes time.Now() straight through.
		raw := base.Add(time.Duration(nanos))
		a.NoError(s.SetPendingMFAState(ctx, user.ID, "PENDINGSECRET", []string{"code-1", "code-2"}, raw))

		promoted, err := s.UpdateUserMFAConfigIfPending(ctx, user.ID, raw, &storepb.MFAConfig{
			OtpSecret:     "PENDINGSECRET",
			RecoveryCodes: []string{"code-1", "code-2"},
		})
		a.NoError(err)
		a.NotNil(promoted, "a confirmation carrying the version it was minted with must promote, nanos=%d", nanos)
		a.Equal("PENDINGSECRET", promoted.MFAConfig.GetOtpSecret(), "nanos=%d", nanos)

		// The row reports the instant it can actually hold, so a handler that
		// mints the same way compares equal to it.
		a.NoError(s.SetPendingMFAState(ctx, user.ID, "AGAIN", []string{"code-3"}, raw))
		reread, err := s.GetUserByID(ctx, user.ID)
		a.NoError(err)
		stored := reread.MFAConfig.GetTempOtpSecretCreatedTime().AsTime()
		a.True(stored.Equal(raw.Truncate(time.Microsecond)), "stored %v, minted %v", stored, raw)
	}
}
