package redis

import (
	"strings"

	"github.com/google/shlex"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

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
	"hget":                 true,
	"hgetall":              true,
	"hkeys":                true,
	"hmget":                true,
	"hlen":                 true,
	"hrandfield":           true,
	"hscan":                true,
	"hvals":                true,
	"hstrlen":              true,
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
	"select":               true,
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
	if _, ok := readCommands[command]; ok {
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
