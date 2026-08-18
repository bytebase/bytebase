package redis

import (
	"strings"

	"github.com/google/shlex"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// readCommands holds the core Redis commands that the server flags "readonly"
// in its own command table. Redis has no read-only transaction or connection
// mode, so the SQL Editor read-only gate has to classify by command name.
//
// TestReadCommandsMatchServerFlags diffs this map against
// testdata/core_readonly_commands.txt, a snapshot of `COMMAND INFO` from the
// pinned Redis in that file's header. Regenerate the snapshot when bumping the
// pinned version; the test then reports every newly shipped read command.
// Commands left out on purpose live in that test's knownExclusions, with the
// reason.
var readCommands = map[string]bool{
	"bitcount":             true,
	"bitfield_ro":          true,
	"bitpos":               true,
	"dbsize":               true,
	"dump":                 true,
	"exists":               true,
	"expiretime":           true,
	"get":                  true,
	"getrange":             true,
	"geodist":              true,
	"geohash":              true,
	"georadius_ro":         true,
	"georadiusbymember_ro": true,
	"geopos":               true,
	"geosearch":            true,
	"getbit":               true,
	"hexists":              true,
	"hexpiretime":          true,
	"hget":                 true,
	"hgetall":              true,
	"hkeys":                true,
	"hmget":                true,
	"hlen":                 true,
	"hpexpiretime":         true,
	"hpttl":                true,
	"hrandfield":           true,
	"hscan":                true,
	"hvals":                true,
	"hstrlen":              true,
	"httl":                 true,
	"keys":                 true,
	"lcs":                  true,
	"lindex":               true,
	"llen":                 true,
	"lrange":               true,
	"lolwut":               true,
	"lpos":                 true,
	"mget":                 true,
	"pfcount":              true,
	"pttl":                 true,
	"pexpiretime":          true,
	"randomkey":            true,
	"scan":                 true,
	"sscan":                true,
	"scard":                true,
	"sdiff":                true,
	"smismember":           true,
	"sismember":            true,
	"sinter":               true,
	"sintercard":           true,
	"smembers":             true,
	"sort_ro":              true,
	"srandmember":          true,
	"substr":               true,
	"sunion":               true,
	"strlen":               true,
	"ttl":                  true,
	"touch":                true,
	"type":                 true,
	"xpending":             true,
	"xrange":               true,
	"xread":                true,
	"xrevrange":            true,
	"xlen":                 true,
	"zcard":                true,
	"zcount":               true,
	"zdiff":                true,
	"zinter":               true,
	"zintercard":           true,
	"zlexcount":            true,
	"zmscore":              true,
	"zrange":               true,
	"zrangebyscore":        true,
	"zrangebylex":          true,
	"zrandmember":          true,
	"zrank":                true,
	"zrevrank":             true,
	"zrevrange":            true,
	"zrevrangebylex":       true,
	"zrevrangebyscore":     true,
	"zscore":               true,
	"zscan":                true,
	"zunion":               true,
}

var doubleNameReadCommands = map[string]map[string]bool{
	"xinfo": {
		"groups":    true,
		"stream":    true,
		"consumers": true,
	},
	"object": {
		"freq":     true,
		"encoding": true,
		"idletime": true,
		"refcount": true,
	},
	"memory": {
		"usage": true,
	},
}

// moduleReadCommands holds the read-only commands of the Redis Stack modules.
// Module commands namespace with a dot inside a single token ("JSON.GET"), so
// they cannot go through doubleNameReadCommands, which expects the subcommand
// as a second token ("XINFO STREAM").
//
// The server's "readonly" flag means "does not write the keyspace", not "has no
// side effects", so it is not enough on its own here. RediSearch flags
// FT.ALIASADD, FT.DICTADD, FT.DICTDEL, FT.CONFIG and FT.CURSOR readonly even
// though they mutate index or config state, and RedisBloom flags CF.COMPACT
// readonly even though it rewrites the filter. Those stay out. So do the
// modules' internal cluster commands.
var moduleReadCommands = map[string]bool{
	// RedisJSON.
	"json.arrindex": true,
	"json.arrlen":   true,
	"json.debug":    true,
	"json.get":      true,
	"json.mget":     true,
	"json.objkeys":  true,
	"json.objlen":   true,
	"json.resp":     true,
	"json.strlen":   true,
	"json.type":     true,

	// RedisTimeSeries.
	"ts.get":        true,
	"ts.info":       true,
	"ts.mget":       true,
	"ts.mrange":     true,
	"ts.mrevrange":  true,
	"ts.queryindex": true,
	"ts.range":      true,
	"ts.revrange":   true,

	// RedisBloom: Bloom filter, Cuckoo filter, Count-Min sketch, Top-K,
	// t-digest. The scandump commands stay out: they page through internal
	// chunks for replication, not for reading data.
	"bf.card":              true,
	"bf.exists":            true,
	"bf.info":              true,
	"bf.mexists":           true,
	"cf.count":             true,
	"cf.exists":            true,
	"cf.info":              true,
	"cf.mexists":           true,
	"cms.info":             true,
	"cms.query":            true,
	"topk.info":            true,
	"topk.list":            true,
	"topk.query":           true,
	"tdigest.byrank":       true,
	"tdigest.byrevrank":    true,
	"tdigest.cdf":          true,
	"tdigest.info":         true,
	"tdigest.max":          true,
	"tdigest.min":          true,
	"tdigest.quantile":     true,
	"tdigest.rank":         true,
	"tdigest.revrank":      true,
	"tdigest.trimmed_mean": true,

	// RediSearch. FT.AGGREGATE with WITHCURSOR leaves a server-side cursor
	// behind, and FT.CURSOR is not allowed here, so such a cursor is only
	// reclaimed when it times out.
	"ft.aggregate":  true,
	"ft.dictdump":   true,
	"ft.explain":    true,
	"ft.get":        true,
	"ft.info":       true,
	"ft.mget":       true,
	"ft.profile":    true,
	"ft.search":     true,
	"ft.spellcheck": true,
	"ft.sugget":     true,
	"ft.suglen":     true,
	"ft.syndump":    true,
	"ft.tagvals":    true,
}

func init() {
	base.RegisterQueryValidator(storepb.Engine_REDIS, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	lines := strings.Split(statement, "\n")
	for _, line := range lines {
		fields, err := shlex.Split(line)
		if err != nil {
			return false, false, errors.Wrapf(err, "failed to split command %s", line)
		}
		if len(fields) == 0 {
			continue
		}
		if !isReadCommand(fields) {
			return false, false, nil
		}
	}
	return true, true, nil
}

func isReadCommand(fields []string) bool {
	if len(fields) == 0 {
		return false
	}
	command := strings.ToLower(fields[0])
	if readCommands[command] || moduleReadCommands[command] {
		return true
	}
	if doubleNameReadCommands[command] != nil && len(fields) > 1 {
		if d, ok := doubleNameReadCommands[command]; ok && d != nil {
			if _, ok := d[strings.ToLower(fields[1])]; ok {
				return true
			}
		}
	}
	return false
}
