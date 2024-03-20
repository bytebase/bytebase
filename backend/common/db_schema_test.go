package common

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassification(t *testing.T) {
	tests := []struct {
		rawComment     string
		classification string
		userComment    string
		rebuildComment string
		workspaceID    string
	}{
		{
			rawComment:     "1abc",
			classification: "",
			userComment:    "1abc",
			rebuildComment: "1abc",
		},
		{
			rawComment:     "abc",
			classification: "",
			userComment:    "abc",
			rebuildComment: "abc",
		},
		{
			rawComment:     "-abc",
			classification: "",
			userComment:    "-abc",
			rebuildComment: "-abc",
		},
		{
			rawComment:     "0-abc",
			classification: "0",
			userComment:    "abc",
			rebuildComment: "0-abc",
		},
		{
			rawComment:     "1-2-abc",
			classification: "1-2",
			userComment:    "abc",
			rebuildComment: "1-2-abc",
		},
		{
			rawComment:     "1-2 abc",
			classification: "1-2",
			userComment:    "abc",
			rebuildComment: "1-2-abc",
			workspaceID:    "a4fea42c-c097-47c3-b661-4a6fcea1cf6d",
		},
		{
			rawComment:     "1 abc",
			classification: "1",
			userComment:    "abc",
			rebuildComment: "1-abc",
			workspaceID:    "a4fea42c-c097-47c3-b661-4a6fcea1cf6d",
		},
		{
			rawComment:     "1",
			classification: "1",
			userComment:    "",
			rebuildComment: "1",
		},
		{
			rawComment:     "1- 2",
			classification: "1",
			userComment:    " 2",
			rebuildComment: "1- 2",
		},
		{
			rawComment:     "1 2",
			classification: "1",
			userComment:    "2",
			rebuildComment: "1-2",
			workspaceID:    "a4fea42c-c097-47c3-b661-4a6fcea1cf6d",
		},
		{
			rawComment:     "1-2",
			classification: "1-2",
			userComment:    "",
			rebuildComment: "1-2",
		},
		{
			rawComment:     "1-2a",
			classification: "1",
			userComment:    "2a",
			rebuildComment: "1-2a",
		},
		{
			rawComment:     "1:a",
			classification: "",
			userComment:    "1:a",
			rebuildComment: "1:a",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test classification for comment: %v", test.rawComment), func(t *testing.T) {
			err := os.Setenv("BYTEBASE_WORKSPACE_ID", test.workspaceID)
			assert.NoError(t, err)
			classification, userComment := GetClassificationAndUserComment(test.rawComment)
			assert.Equal(t, test.classification, classification)
			assert.Equal(t, test.userComment, userComment)
			rebuildComment := GetCommentFromClassificationAndUserComment(classification, userComment)
			assert.Equal(t, test.rebuildComment, rebuildComment)
		})
	}
}
