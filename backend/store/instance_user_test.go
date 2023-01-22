package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInstanceUsersDiff(t *testing.T) {
	oldInstanceUsers := []*InstanceUserMessage{
		{Name: "a", Grant: "1"},
		{Name: "b", Grant: "2"},
		{Name: "c", Grant: "3"},
	}
	newInstanceUsers := []*InstanceUserMessage{
		{Name: "b", Grant: "2"},
		{Name: "c", Grant: "8"},
		{Name: "d", Grant: "4"},
	}
	toDelete, upserts := getInstanceUsersDiff(oldInstanceUsers, newInstanceUsers)
	require.Equal(t, []string{"a"}, toDelete)
	require.Equal(t, []*InstanceUserMessage{{Name: "c", Grant: "8"}, {Name: "d", Grant: "4"}}, upserts)
}
