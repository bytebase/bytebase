package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassification(t *testing.T) {
	tests := []struct {
		rawComment     string
		classification string
		userComment    string
	}{
		{
			rawComment:     "1abc",
			classification: "",
			userComment:    "1abc",
		},
		{
			rawComment:     "abc",
			classification: "",
			userComment:    "abc",
		},
		{
			rawComment:     "-abc",
			classification: "",
			userComment:    "-abc",
		},
		{
			rawComment:     "0-abc",
			classification: "0",
			userComment:    "abc",
		},
		{
			rawComment:     "1-2-abc",
			classification: "1-2",
			userComment:    "abc",
		},
		{
			rawComment:     "1",
			classification: "1",
			userComment:    "",
		},
		{
			rawComment:     "1-2",
			classification: "1-2",
			userComment:    "",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test classification for comment: %v", test.rawComment), func(t *testing.T) {
			classification, userComment := GetClassificationAndUserComment(test.rawComment)
			assert.Equal(t, test.classification, classification)
			assert.Equal(t, test.userComment, userComment)
			rawComment := GetCommentFromClassificationAndUserComment(classification, userComment)
			assert.Equal(t, test.rawComment, rawComment)
		})
	}
}
