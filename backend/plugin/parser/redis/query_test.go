package redis

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		statement string
		valid     bool
		allQuery  bool
		err       bool
	}{
		{
			statement: "get hello",
			valid:     true,
			allQuery:  true,
		},
		{
			statement: "set hello 1",
			valid:     false,
			allQuery:  false,
		},
		{
			statement: "memory usage hello",
			valid:     true,
			allQuery:  true,
		},
		// RedisJSON reads. The command namespaces with a dot inside one token,
		// so it can only match readCommands/moduleReadCommands, never the
		// container-subcommand fallback.
		{
			statement: "JSON.GET doc",
			valid:     true,
			allQuery:  true,
		},
		{
			statement: "json.get doc $.name",
			valid:     true,
			allQuery:  true,
		},
		{
			statement: "JSON.DEBUG MEMORY doc",
			valid:     true,
			allQuery:  true,
		},
		{
			statement: `JSON.SET doc $ {"a":1}`,
			valid:     false,
			allQuery:  false,
		},
		{
			statement: "JSON.DEL doc $.a",
			valid:     false,
			allQuery:  false,
		},
		// Other Redis Stack modules.
		{
			statement: "FT.SEARCH idx *",
			valid:     true,
			allQuery:  true,
		},
		{
			statement: "TS.GET series",
			valid:     true,
			allQuery:  true,
		},
		{
			statement: "BF.EXISTS filter item",
			valid:     true,
			allQuery:  true,
		},
		// FT.DICTADD mutates a dictionary even though Redis flags it readonly.
		{
			statement: "FT.DICTADD dict term",
			valid:     false,
			allQuery:  false,
		},
		// Core read commands added in Redis 7.4.
		{
			statement: "HTTL h FIELDS 1 f",
			valid:     true,
			allQuery:  true,
		},
		// Read-only script mode only blocks commands flagged "write", so Lua
		// still reaches SELECT and the readonly-flagged module commands that
		// mutate state. See knownExclusions in
		// TestReadCommandsMatchServerFlags.
		{
			statement: "EVAL_RO \"return redis.call('GET','k')\" 0",
			valid:     false,
			allQuery:  false,
		},
		{
			statement: "EVALSHA_RO abc 0",
			valid:     false,
			allQuery:  false,
		},
		{
			statement: "FCALL_RO fn 0",
			valid:     false,
			allQuery:  false,
		},
		// SELECT switches the connection to another logical database, so a
		// later read escapes the database the request was authorized against.
		{
			statement: "select 3",
			valid:     false,
			allQuery:  false,
		},
		{
			statement: "select 3\nkeys *",
			valid:     false,
			allQuery:  false,
		},
	}

	for _, test := range tests {
		gotValid, gotAllQuery, err := validateQuery(test.statement)
		if test.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.valid, gotValid, test.statement)
			require.Equal(t, test.allQuery, gotAllQuery, test.statement)
		}
	}
}

// TestReadCommandsMatchServerFlags diffs the core allowlist against a snapshot
// of the flags Redis reports for itself, so a Redis upgrade that ships new read
// commands fails here instead of reaching a customer. See the snapshot header
// for how to regenerate it.
func TestReadCommandsMatchServerFlags(t *testing.T) {
	// knownExclusions holds commands Redis flags "readonly" that Bytebase
	// deliberately keeps out of the gate. Every entry needs a reason: the flag
	// means "does not write the keyspace", not "has no side effects".
	//
	// SELECT is not listed. Redis does not flag it readonly, so the reverse
	// check below is what keeps it out of the allowlist.
	knownExclusions := map[string]string{
		// Read-only script mode only rejects commands flagged "write", so a
		// script still reaches SELECT and every readonly-flagged module command
		// that mutates state. Verified on redis-stack-server 7.4.7: running
		// EVAL_RO "redis.call('SELECT',1); return redis.call('GET','marker')"
		// against database 0 returns database 1's value, and a no-writes
		// function called through FCALL_RO both crosses databases and adds a
		// RediSearch dictionary term via FT.DICTADD.
		"eval_ro":    "Lua reaches SELECT and readonly-flagged mutating module commands",
		"evalsha_ro": "same as eval_ro",
		"fcall_ro":   "same as eval_ro, through a no-writes function",
	}

	content, err := os.ReadFile("testdata/core_readonly_commands.txt")
	require.NoError(t, err)

	serverReadOnly := map[string]bool{}
	for line := range strings.SplitSeq(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		serverReadOnly[line] = true
	}
	require.NotEmpty(t, serverReadOnly, "snapshot has no commands")

	allowed := map[string]bool{}
	for command := range readCommands {
		allowed[command] = true
	}
	for container, subcommands := range doubleNameReadCommands {
		for subcommand := range subcommands {
			allowed[container+" "+subcommand] = true
		}
	}

	var rejected []string
	for command := range serverReadOnly {
		if allowed[command] {
			continue
		}
		if _, ok := knownExclusions[command]; ok {
			continue
		}
		rejected = append(rejected, command)
	}
	slices.Sort(rejected)
	// assert, not require: a run that breaks both directions must report both.
	assert.Emptyf(t, rejected,
		"Redis flags these commands readonly but the gate rejects them: %v. "+
			"Add each to readCommands or doubleNameReadCommands, or to knownExclusions with a reason.",
		rejected)

	var unflagged []string
	for command := range allowed {
		if !serverReadOnly[command] {
			unflagged = append(unflagged, command)
		}
	}
	slices.Sort(unflagged)
	assert.Emptyf(t, unflagged,
		"the gate accepts these but Redis does not flag them readonly: %v. "+
			"Either the name is wrong, or the command has side effects and must be removed.",
		unflagged)
}

// TestModuleReadCommandsWellFormed guards the module allowlist, which has no
// snapshot because module presence varies per instance. A key that is not a
// lowercase dotted name can never match, since isReadCommand lowercases the
// first token and module commands carry the dot inside it.
func TestModuleReadCommandsWellFormed(t *testing.T) {
	require.NotEmpty(t, moduleReadCommands)
	for command := range moduleReadCommands {
		require.Equal(t, strings.ToLower(command), command, "module command must be lowercase")
		require.Contains(t, command, ".", "module command must be dot-namespaced")
		require.False(t, readCommands[command], "module command duplicated in readCommands: %s", command)
	}
}
