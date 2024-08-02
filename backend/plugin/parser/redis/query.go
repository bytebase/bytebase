package redis

import (
	"strings"

	"github.com/google/shlex"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var readCommands = map[string]bool{
	"exists":        true,
	"get":           true,
	"hexists":       true,
	"hget":          true,
	"hgetall":       true,
	"hkeys":         true,
	"hmget":         true,
	"hlen":          true,
	"hvals":         true,
	"lindex":        true,
	"llen":          true,
	"lrange":        true,
	"pttl":          true,
	"scan":          true,
	"scard":         true,
	"sdiff":         true,
	"select":        true,
	"sismember":     true,
	"smembers":      true,
	"sunion":        true,
	"strlen":        true,
	"ttl":           true,
	"type":          true,
	"zcard":         true,
	"zcount":        true,
	"zrange":        true,
	"zrangebyscore": true,
	"zrank":         true,
	"zscore":        true,
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
		command := strings.ToLower(fields[0])
		if !readCommands[command] {
			return false, false, nil
		}
	}
	return true, true, nil
}
