package vcs

import (
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestIsDoubleTimesAsteriskInTemplateValid(t *testing.T) {
	tests := []struct {
		template string
		err      bool
	}{
		{
			template: "**",
			err:      true,
		},
		{
			template: "bytebase/{{ENV_ID}}/**",
			err:      true,
		},
		{
			template: "**/{{ENV_ID}}/{{DB_NAME}}.sql",
			err:      true,
		},
		{
			template: "bytebase/**/{{ENV_ID}}/**/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			err:      false,
		},
		{
			template: "/**/{{ENV_ID}}/**/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			err:      false,
		},
		// Credit to Linear Issue BYT-1267
		{
			template: "/configure/configure/{{ENV_ID}}/**/**/{{DESCRIPTION}}.sql",
			err:      false,
		},
	}
	for _, test := range tests {
		outputErr := isDoubleAsteriskInTemplateValid(test.template)
		if test.err {
			assert.Error(t, outputErr)
		} else {
			assert.NoError(t, outputErr)
		}
	}
}

func TestPushEventUnmarshalToProto(t *testing.T) {
	a := require.New(t)
	vcsPushEvent := &PushEvent{
		VCSType:            GitLab,
		BaseDirectory:      "aaa",
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
		FileCommit: FileCommit{
			ID:          "1",
			Title:       "123",
			Message:     "file",
			CreatedTs:   123,
			URL:         "terw",
			AuthorName:  "hi",
			AuthorEmail: "none",
			Added:       "aaa",
		},
	}
	bytes, err := json.Marshal(vcsPushEvent)
	a.NoError(err)
	var pushEvent storepb.PushEvent
	err = protojson.Unmarshal(bytes, &pushEvent)
	a.NoError(err)
	a.Equal(storepb.VcsType_GITLAB, pushEvent.VcsType)

	p := &storepb.InstanceChangeHistoryPayload{
		PushEvent: &pushEvent,
	}
	bytes, err = protojson.Marshal(p)
	a.NoError(err)
	t.Logf("%q", string(bytes))
}
