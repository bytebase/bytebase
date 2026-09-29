package aireview

import (
	"context"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"golang.org/x/text/cases"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	toolSearch = "search"
	toolRead   = "read"

	// maxReadObjects is how many names one read takes. The names come as a
	// batch because not every endpoint emits parallel tool calls, and one
	// object per call would cost one model call per object.
	maxReadObjects = 20
	// maxToolResultBytes is where a result is cut and paged, so that one call
	// cannot fill the model's context. A definition cut in the middle is
	// measured before JSON escaping, so that result can run somewhat over it.
	maxToolResultBytes = 32 << 10
)

const (
	kindTable            = "table"
	kindExternalTable    = "external table"
	kindView             = "view"
	kindMaterializedView = "materialized view"
	kindFunction         = "function"
	kindProcedure        = "procedure"
	kindSequence         = "sequence"
	kindPackage          = "package"
	kindEvent            = "event"
	kindStream           = "stream"
	kindTask             = "task"
	kindEnumType         = "enum type"
	kindCompositeType    = "composite type"
	kindEventTrigger     = "event trigger"
	kindExtension        = "extension"
)

// catalogTools answers search and read from the synced schema of one
// database. It reads nothing live, so a fact the sync does not hold is a fact
// the model cannot get.
type catalogTools struct {
	engine storepb.Engine
	schema *metadatapb.DatabaseSchemaMetadata

	// mu guards the objects, their definitions, and the last search.
	mu      sync.Mutex
	indexed bool
	objects []*catalogObject
	// searched keeps the matches of the last search, which its next page
	// asks for again.
	searched *searchArguments
	matches  []searchMatch
}

// NewCatalogTools returns the search and read tools over a database's synced
// schema.
func NewCatalogTools(engine storepb.Engine, schema *metadatapb.DatabaseSchemaMetadata) Tools {
	return &catalogTools{engine: engine, schema: schema}
}

// catalogObject is anything in the database that has a definition of its own.
type catalogObject struct {
	kind   string
	schema string
	name   string
	// signature tells the overloads of a routine apart.
	signature  string
	statistics *tableStatistics

	// render builds the definition on its first use, so that reading one
	// table of a large database does not render the others.
	render     func() string
	definition *string
}

func (o *catalogObject) text() string {
	if o.definition == nil {
		definition := strings.TrimSpace(o.render())
		o.definition = &definition
		o.render = nil
	}
	return *o.definition
}

type tableStatistics struct {
	Rows       int64 `json:"rows"`
	DataBytes  int64 `json:"data_bytes"`
	IndexBytes int64 `json:"index_bytes"`
}

type objectName struct {
	Schema string `json:"schema,omitempty"`
	Name   string `json:"name"`
}

type searchArguments struct {
	Text   string `json:"text"`
	Schema string `json:"schema"`
	From   int    `json:"from"`
}

type searchMatch struct {
	Kind      string `json:"kind"`
	Schema    string `json:"schema,omitempty"`
	Name      string `json:"name"`
	Signature string `json:"signature,omitempty"`
	// Matched says where the text was found: name or definition.
	Matched string `json:"matched"`
}

type searchResult struct {
	Matches  []searchMatch `json:"matches"`
	Total    int           `json:"total"`
	NextFrom *int          `json:"next_from,omitempty"`
	Note     string        `json:"note,omitempty"`
}

type readArguments struct {
	Objects []objectName `json:"objects"`
	From    int          `json:"from"`
}

type readObject struct {
	Kind       string `json:"kind"`
	Schema     string `json:"schema,omitempty"`
	Name       string `json:"name"`
	Signature  string `json:"signature,omitempty"`
	Definition string `json:"definition"`
	// Part says which bytes of the definition these are, when not all of them.
	Part       string           `json:"part,omitempty"`
	Statistics *tableStatistics `json:"statistics,omitempty"`
}

type readResult struct {
	Objects  []readObject `json:"objects"`
	NotFound []objectName `json:"not_found,omitempty"`
	NextFrom *int         `json:"next_from,omitempty"`
	Note     string       `json:"note,omitempty"`
}

// Definitions implements Tools.
func (*catalogTools) Definitions() []*v1pb.AIChatToolDefinition {
	return []*v1pb.AIChatToolDefinition{
		{
			Name:             toolSearch,
			Description:      "Find the objects of the database whose name or definition contains the text, matched as a case insensitive substring. Use it to find what depends on an object: the views that read a table, the routines that update it, the triggers that write to it. An empty text lists every object. When the result is cut it carries next_from; call search again with the same arguments and from set to it.",
			ParametersSchema: `{"type": "object", "properties": {"text": {"type": "string", "description": "The text to look for in names and definitions."}, "schema": {"type": "string", "description": "Limit the search to this schema."}, "from": {"type": "integer", "description": "Where to continue a cut result, taken from next_from."}}, "required": ["text"]}`,
		},
		{
			Name:             toolRead,
			Description:      fmt.Sprintf("Return the definition and statistics of the objects of up to %d names. A table comes with its columns, indexes, constraints, and triggers. Its statistics are the row count, which is the engine's estimate, and the sizes in bytes. A table without statistics has none synced, which does not say it is empty. A name returns every object of that name, such as every overload of a function, and without a schema it returns them from every schema. When the result is cut it carries next_from; call read again with the same objects and from set to it.", maxReadObjects),
			ParametersSchema: `{"type": "object", "properties": {"objects": {"type": "array", "description": "The names to read.", "items": {"type": "object", "properties": {"schema": {"type": "string"}, "name": {"type": "string"}}, "required": ["name"]}}, "from": {"type": "integer", "description": "Where to continue a cut result, taken from next_from."}}, "required": ["objects"]}`,
		},
	}
}

// Call implements Tools. Arguments the model got wrong come back as text it
// can correct. Only a failure of the tool itself is an error.
func (c *catalogTools) Call(_ context.Context, name string, arguments string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch name {
	case toolSearch:
		return c.search(arguments)
	case toolRead:
		return c.read(arguments)
	default:
		return "", errors.Errorf("tool %q does not exist", name)
	}
}

func (c *catalogTools) search(arguments string) (string, error) {
	var args searchArguments
	if err := jsonv2.Unmarshal([]byte(arguments), &args); err != nil {
		return fmt.Sprintf(`invalid arguments: %v; pass {"text": "orders", "schema": "public", "from": 0}, where schema and from are optional`, err), nil
	}
	if args.From < 0 {
		return "from must be 0 or greater", nil
	}

	matches := c.match(args.Text, args.Schema)
	result := &searchResult{Matches: []searchMatch{}, Total: len(matches)}
	size := 0
	for i := args.From; i < len(matches); i++ {
		row, err := encode(matches[i])
		if err != nil {
			return "", err
		}
		if size+len(row) > maxToolResultBytes && len(result.Matches) > 0 {
			next := i
			result.NextFrom = &next
			result.Note = fmt.Sprintf("cut at match %d of %d; call search again with the same arguments and from set to %d", i, len(matches), i)
			break
		}
		size += len(row)
		result.Matches = append(result.Matches, matches[i])
	}
	return encode(result)
}

// match returns every object that matches the text, in the order of the index.
func (c *catalogTools) match(text string, schemaName string) []searchMatch {
	if c.searched != nil && c.searched.Text == text && c.searched.Schema == schemaName {
		return c.matches
	}
	needle := fold(text)
	inSchema := c.schemaFilter(schemaName)
	var matches []searchMatch
	for _, object := range c.index() {
		if !inSchema(object) {
			continue
		}
		matched := ""
		switch {
		case strings.Contains(fold(object.name), needle):
			matched = "name"
		case strings.Contains(fold(object.text()), needle):
			matched = "definition"
		default:
			continue
		}
		matches = append(matches, searchMatch{Kind: object.kind, Schema: object.schema, Name: object.name, Signature: object.signature, Matched: matched})
	}
	c.searched, c.matches = &searchArguments{Text: text, Schema: schemaName}, matches
	return matches
}

func (c *catalogTools) read(arguments string) (string, error) {
	var args readArguments
	if err := jsonv2.Unmarshal([]byte(arguments), &args); err != nil {
		return fmt.Sprintf(`invalid arguments: %v; pass {"objects": [{"schema": "public", "name": "orders"}], "from": 0}, where schema and from are optional`, err), nil
	}
	switch {
	case len(args.Objects) == 0:
		return `pass at least one object: {"objects": [{"schema": "public", "name": "orders"}]}`, nil
	case len(args.Objects) > maxReadObjects:
		return fmt.Sprintf("read takes at most %d names in one call and you passed %d; split them over several calls", maxReadObjects, len(args.Objects)), nil
	case args.From < 0:
		return "from must be 0 or greater", nil
	}

	result := &readResult{Objects: []readObject{}}
	var found []*catalogObject
	seen := make(map[*catalogObject]bool)
	for _, want := range args.Objects {
		objects := c.lookup(want)
		if len(objects) == 0 {
			result.NotFound = append(result.NotFound, want)
			continue
		}
		for _, object := range objects {
			if !seen[object] {
				seen[object] = true
				found = append(found, object)
			}
		}
	}

	// The definitions of the objects found, laid end to end, are one text, and
	// from is a position in it. A name alone cannot say where to continue,
	// since the overloads of a function share theirs. Every definition takes
	// one position more than its bytes, so that an empty one has a position
	// of its own and a cursor behind it has moved.
	size, end := 0, 0
	for _, object := range found {
		definition := object.text()
		start := end
		end += len(definition) + 1
		if end <= args.From {
			continue
		}
		begin := runeStart(definition, max(args.From-start, 0))
		if begin > 0 && begin == len(definition) {
			continue
		}
		entry := readObject{
			Kind:       object.kind,
			Schema:     object.schema,
			Name:       object.name,
			Signature:  object.signature,
			Definition: definition[begin:],
			Part:       part(begin, len(definition), len(definition)),
			Statistics: object.statistics,
		}
		// The encoding is never shorter than the text, so a text over the
		// limit is not encoded to find that out.
		if len(entry.Definition) <= maxToolResultBytes {
			encoded, err := encode(entry)
			if err != nil {
				return "", err
			}
			size += len(encoded)
			if size <= maxToolResultBytes {
				result.Objects = append(result.Objects, entry)
				continue
			}
		}
		if len(result.Objects) > 0 {
			// A definition is cut only when it does not fit a result of its own.
			next := start + begin
			result.NextFrom = &next
			break
		}
		stop := cutText(definition, begin, maxToolResultBytes)
		entry.Definition = definition[begin:stop]
		entry.Part = part(begin, stop, len(definition))
		result.Objects = append(result.Objects, entry)
		if stop < len(definition) {
			next := start + stop
			result.NextFrom = &next
			break
		}
	}
	switch {
	case result.NextFrom != nil:
		result.Note = fmt.Sprintf("the result was cut; call read again with the same objects and from set to %d", *result.NextFrom)
	case len(found) > 0 && len(result.Objects) == 0:
		result.Note = fmt.Sprintf("the result ends at %d, so nothing starts at %d", end, args.From)
	default:
	}
	return encode(result)
}

// part names the bytes of a definition a result holds, or nothing when it
// holds them all.
func part(begin int, stop int, size int) string {
	if begin == 0 && stop == size {
		return ""
	}
	return fmt.Sprintf("bytes %d to %d of %d", begin, stop, size)
}

// cutText returns where a piece of text that starts at begin stops: after at
// most budget bytes, behind the last line break among them when there is one,
// and never inside a character.
func cutText(text string, begin int, budget int) int {
	if len(text)-begin <= budget {
		return len(text)
	}
	stop := begin + budget
	if i := strings.LastIndexByte(text[begin:stop], '\n'); i >= 0 {
		return begin + i + 1
	}
	// A character is at most UTFMax bytes, so a longer run without a start is
	// not text and has no character to keep whole.
	for back := 0; back < utf8.UTFMax && stop > begin && !utf8.RuneStart(text[stop]); back++ {
		stop--
	}
	if stop == begin {
		// The budget is under one character, which is taken whole.
		_, size := utf8.DecodeRuneInString(text[begin:])
		return begin + size
	}
	return stop
}

// runeStart moves an offset forward to the start of a character, since a
// model can pass a from of its own making.
func runeStart(text string, offset int) int {
	for offset < len(text) && !utf8.RuneStart(text[offset]) {
		offset++
	}
	return offset
}

// fold maps a text to the form in which texts that differ only in case are
// equal. Lower case is that form for ASCII. Beyond ASCII it is not: the two
// lower case forms of the Greek sigma stay apart.
func fold(text string) string {
	for i := range len(text) {
		if text[i] >= utf8.RuneSelf {
			return cases.Fold().String(text)
		}
	}
	return strings.ToLower(text)
}

// encode allows invalid UTF-8 because a tool failure fails the review, and a
// stray byte in a definition is not worth that.
func encode(value any) (string, error) {
	encoded, err := jsonv2.Marshal(value, jsontext.AllowInvalidUTF8(true))
	if err != nil {
		return "", errors.Wrapf(err, "failed to encode the tool result")
	}
	return string(encoded), nil
}

// lookup finds the objects of a name in the schema asked for. A name that
// matches exactly wins over ones that differ in case, since engines disagree
// on how they fold identifiers and a quoted identifier is not folded at all.
func (c *catalogTools) lookup(want objectName) []*catalogObject {
	inSchema := c.schemaFilter(want.Schema)
	var exact, folded []*catalogObject
	for _, object := range c.index() {
		switch {
		case !inSchema(object):
		case object.name == want.Name:
			exact = append(exact, object)
		case strings.EqualFold(object.name, want.Name):
			folded = append(folded, object)
		default:
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return folded
}

// schemaFilter returns the test for the schema the model asked for. A schema
// of exactly that name wins over the ones that differ in case.
func (c *catalogTools) schemaFilter(schemaName string) func(*catalogObject) bool {
	if schemaName == "" || c.schemaless() {
		return func(*catalogObject) bool { return true }
	}
	for _, s := range c.schema.GetSchemas() {
		if s.GetName() == schemaName {
			return func(object *catalogObject) bool { return object.schema == schemaName }
		}
	}
	return func(object *catalogObject) bool { return strings.EqualFold(object.schema, schemaName) }
}

// schemaless reports an engine without schemas. It syncs one schema with no
// name, and a schema argument there, usually the database name, is ignored.
func (c *catalogTools) schemaless() bool {
	schemas := c.schema.GetSchemas()
	return len(schemas) == 1 && schemas[0].GetName() == ""
}
