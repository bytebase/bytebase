package result

import (
	"strings"
	"sync"
)

const (
	// maxDatabaseSQL caps the SQL text one database's declarative file keeps.
	// A database past it is counted in place of its statements.
	maxDatabaseSQL = 1 << 20
	// maxResultSQL caps the distinct SQL text of every declarative file in the
	// result. Past it, no statement is shown.
	maxResultSQL = 16 << 20
)

// StatementPool holds the text of the statements declarative files ran, each
// distinct text once for the whole result, so that databases running the
// same generated SQL share its memory. Each distinct text gets an id, so
// scripts compare by id and not by text. The logs of one result must share
// one pool. It is safe for concurrent use.
//
// Both caps give the same answer whatever order the logs are read in: one
// depends on a single database, and the other on a total that only grows.
type StatementPool struct {
	mu   sync.Mutex
	text map[string]pooledText
	size int
	full bool
}

type pooledText struct {
	text string
	id   int
}

func NewStatementPool() *StatementPool {
	return &StatementPool{text: map[string]pooledText{}}
}

// Full reports whether the distinct text passed maxResultSQL.
func (p *StatementPool) Full() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.full
}

// sdlStatement is one statement a declarative file ran.
type sdlStatement struct {
	text string
	// id is the same for the same text across the pool's logs.
	id int
	// rows is nil when the command reported no count or its count does not
	// stand, as when it was rolled back.
	rows *int64
}

// keep returns the statements of a declarative file's final attempt. It
// returns none and true when they pass maxDatabaseSQL.
func (p *StatementPool) keep(commands []*Command) ([]sdlStatement, bool) {
	final, _ := finalAttempt(commands, 0)
	var size int
	for _, command := range final {
		size += len(strings.TrimSpace(command.Statement))
	}
	if size > maxDatabaseSQL {
		return nil, true
	}
	statements := make([]sdlStatement, 0, len(final))
	for _, command := range final {
		pooled, ok := p.intern(strings.TrimSpace(command.Statement))
		if !ok {
			return nil, false
		}
		statements = append(statements, sdlStatement{text: pooled.text, id: pooled.id, rows: standingRows(command)})
	}
	return statements, false
}

func (p *StatementPool) intern(text string) (pooledText, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.full {
		return pooledText{}, false
	}
	if pooled, ok := p.text[text]; ok {
		return pooled, true
	}
	if p.size+len(text) > maxResultSQL {
		p.full, p.text = true, nil
		return pooledText{}, false
	}
	pooled := pooledText{text: text, id: len(p.text)}
	p.text[text] = pooled
	p.size += len(text)
	return pooled, true
}

// standingRows is the command's count when it stands: reported, committed
// and not rolled back.
func standingRows(command *Command) *int64 {
	if command.AffectedRows == nil || command.CommitUnknown || command.RolledBack {
		return nil
	}
	return command.AffectedRows
}
