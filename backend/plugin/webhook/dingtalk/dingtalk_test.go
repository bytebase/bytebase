package dingtalk

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

func TestGetDingTalkMobileFromUserStripsCountryCode(t *testing.T) {
	a := require.New(t)

	got, err := getDingTalkMobileFromUser(&store.UserMessage{Phone: "+8613800000000"})
	a.NoError(err)
	a.Equal("13800000000", got)
}

func TestGetDingTalkMobileFromPhoneStripsCountryCode(t *testing.T) {
	a := require.New(t)

	got, err := getDingTalkMobileFromPhone("+8613800000000")
	a.NoError(err)
	a.Equal("13800000000", got)
}
