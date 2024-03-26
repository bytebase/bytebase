package vcs

import (
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestPushEventUnmarshalToProto(t *testing.T) {
	a := require.New(t)
	vcsPushEvent := &PushEvent{
		VCSType:            GitLab,
		Ref:                "refs/heads/master",
		Before:             "beforea",
		After:              "afterb",
		RepositoryID:       "id-123",
		RepositoryURL:      "sptth://123",
		RepositoryFullPath: "utlenme",
		AuthorName:         "me",
		CommitList: []Commit{
			{
				ID:           "1",
				Title:        "123",
				Message:      "file",
				CreatedTs:    123,
				URL:          "terw",
				AuthorName:   "hi",
				AuthorEmail:  "none",
				AddedList:    []string{"123"},
				ModifiedList: []string{"321"},
			},
		},
	}
	bytes, err := json.Marshal(vcsPushEvent)
	a.NoError(err)
	var pushEvent storepb.PushEvent
	err = protojson.Unmarshal(bytes, &pushEvent)
	a.NoError(err)
	a.Equal(storepb.VcsType_GITLAB, pushEvent.VcsType)
}
