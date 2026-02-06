package elasticsearch

import (
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	hjson "github.com/hjson/hjson-go/v4"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type ParseResult struct {
	Requests []*Request
	Errors   []*base.SyntaxError `yaml:"errors,omitempty"`
}

type Request struct {
	Method      string
	URL         string
	Data        []string `yaml:"data,omitempty"`
	StartOffset int
	EndOffset   int
}

type editorRequest struct {
	method string
	url    string
	data   []string
}

type syntaxError struct {
	// byteOffset is the byte offset of the first character of the error occurred, start by 0.
	byteOffset int
	message    string
}

// parsedRequest is the range of a request, it is left inclusive and right exclusive.
type parsedRequest struct {
	// startOffset is the byte offset of the first character of the request.
	startOffset int
	// endOffset is the byte offset of the end position.
	endOffset int
}

type adjustedParsedRequest struct {
	// startLineNumber is the line number of the first character of the request, starting from 0.
	startLineNumber int
	// endLineNumber is the line number of the end position, starting from 0.
	endLineNumber int
}

// See https://sourcegraph.com/github.com/elastic/kibana/-/blob/src/platform/plugins/shared/console/public/application/containers/editor/utils/requests_utils.ts?L76.
// Combine getRequestStartLineNumber and getRequestEndLineNumber.
func getAdjustedParsedRequest(r parsedRequest, text string, nextRequest *parsedRequest) adjustedParsedRequest {
	bs := []byte(text)
	startLineNumber := 0
	endLineNumber := 0
	startOffset := r.startOffset
	// The startOffset is out of range, returning the end of document like
	// what the model.getPositionAt does.
	if r.startOffset >= len(bs) {
		startOffset = len(bs) - 1
	}
	for i := 0; i < r.startOffset; i++ {
		if bs[i] == '\n' {
			startLineNumber++
		}
	}

	if r.endOffset >= 0 {
		// if the parser set an end offset for this request , then find the line number for it.
		endLineNumber = startLineNumber
		endOffset := r.endOffset
		if endOffset >= len(bs) {
			endOffset = len(bs) - 1
		}
		for i := startOffset; i < endOffset; i++ {
			if bs[i] == '\n' {
				endLineNumber++
			}
		}
	} else {
		// if no end offset, try to find the line before the next request starts.
		if nextRequest != nil {
			nextRequestStartLine := 0
			nextRequestOffset := nextRequest.startOffset
			if nextRequestOffset >= len(bs) {
				nextRequestOffset = len(bs) - 1
			}
			for i := 0; i < nextRequestOffset; i++ {
				if bs[i] == '\n' {
					nextRequestStartLine++
				}
			}
			if nextRequestStartLine > startLineNumber {
				endLineNumber = nextRequestStartLine - 1
			} else {
				endLineNumber = startLineNumber
			}
		} else {
			// If there is no next request, find the end of the text or the line that starts with a method.
			lines := strings.Split(text, "\n")
			nextLineNumber := 0
			for i := 0; i < r.startOffset; i++ {
				if bs[i] == '\n' {
					nextLineNumber++
				}
			}
			nextLineNumber++
			for nextLineNumber < len(lines) {
				content := strings.TrimSpace(lines[nextLineNumber])
				startsWithMethodRegex := regexp.MustCompile(`(?i)^\s*(GET|POST|PUT|PATCH|DELETE)`)
				if startsWithMethodRegex.MatchString(content) {
					break
				}
				nextLineNumber++
			}
			// nextLineNumber is now either the line with a method or 1 line after the end of the text
			// set the end line for this request to the line before nextLineNumber
			if nextLineNumber > startLineNumber {
				endLineNumber = nextLineNumber - 1
			} else {
				endLineNumber = startLineNumber
			}
		}
	}
	// if the end is empty, go up to find the first non-empty line.
	lines := strings.Split(text, "\n")
	for endLineNumber >= 0 && strings.TrimSpace(lines[endLineNumber]) == "" {
		endLineNumber--
	}
	return adjustedParsedRequest{
		startLineNumber: startLineNumber,
		endLineNumber:   endLineNumber,
	}
}

type parser struct {
	// at is the current rune's byte offset in the input string.
	at       int
	ch       rune
	escapee  map[rune]string
	text     string
	errors   []*syntaxError
	requests []parsedRequest
	// requestStart is the byte offset of the first character of the request.
	requestStartOffset int
	// requestEnd is the byte offset of the first character of the next request.
	requestEndOffset int
}

// ParseElasticsearchREST parses the Elasticsearch REST API request.
func ParseElasticsearchREST(text string) (*ParseResult, error) {
	p := newParser(text)
	parsedResults, err := p.parse()
	if err != nil {
		return nil, err
	}
	var requests []*Request
	for i, parsedResult := range parsedResults {
		// See https://sourcegraph.com/github.com/elastic/kibana/-/blob/src/platform/plugins/shared/console/public/application/containers/editor/monaco_editor_actions_provider.ts?L261.
		var nextRequest *parsedRequest
		if i < len(parsedResults)-1 {
			nextRequest = &parsedResults[i+1]
		}
		adjustedOffset := getAdjustedParsedRequest(parsedResult, text, nextRequest)
		if adjustedOffset.startLineNumber > adjustedOffset.endLineNumber {
			continue
		}
		editorRequest := getEditorRequest(text, adjustedOffset)
		if editorRequest == nil {
			continue
		}
		if len(editorRequest.data) > 0 {
			for i, v := range editorRequest.data {
				if containsComments(v) {
					// parse and stringify to remove comments.
					editorRequest.data[i] = indentData(v)
				}
				editorRequest.data[i] = collapseLiteralString(v)
			}
		}
		requests = append(requests, &Request{
			Method:      editorRequest.method,
			URL:         editorRequest.url,
			Data:        editorRequest.data,
			StartOffset: parsedResult.startOffset,
			EndOffset:   parsedResult.endOffset,
		})
	}

	// Convert Error
	var syntaxErrors []*base.SyntaxError
	slices.SortFunc(p.errors, func(a, b *syntaxError) int {
		if a.byteOffset < b.byteOffset {
			return -1
		}
		if a.byteOffset > b.byteOffset {
			return 1
		}
		return 0
	})

	for i, err := range p.errors {
		line := 0
		column := 0
		pos := 0
		if i > 0 {
			line = int(syntaxErrors[i-1].Position.Line)
			column = int(syntaxErrors[i-1].Position.Column)
			pos = p.errors[i-1].byteOffset
		}
		boundary := p.errors[i].byteOffset
		if boundary >= len(text) {
			boundary = len(text) - 1
		}

		for j := pos; j <= boundary; j++ {
			if text[j] == '\n' {
				if j == boundary {
					// Decorate the \n position instead of the next line.
					column++
					continue
				}
				line++
				column = 0
			} else {
				column++
			}
		}
		syntaxErrors = append(syntaxErrors, &base.SyntaxError{
			Position: &storepb.Position{
				Line:   int32(line),
				Column: int32(column),
			},
			Message: err.message,
		})
	}

	return &ParseResult{
		Requests: requests,
		Errors:   syntaxErrors,
	}, nil
}

func collapseLiteralString(s string) string {
	splitData := strings.Split(s, `"""`)
	for idx := 1; idx < len(splitData)-1; idx += 2 {
		v, err := json.Marshal(splitData[idx])
		if err != nil {
			continue
		}
		splitData[idx] = string(v)
	}
	return strings.Join(splitData, "")
}

func indentData(s string) string {
	v := make(map[string]any)
	if err := hjson.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	m, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	return string(m)
}

func containsComments(s string) bool {
	insideString := false
	var prevR rune
	rs := []rune(s)
	for i, r := range rs {
		nextR := rune(0)
		if i+1 < len(rs) {
			nextR = rs[i+1]
		}

		if !insideString && r == '"' {
			insideString = true
		} else if insideString && r == '"' && prevR != '\\' {
			insideString = false
		} else if !insideString {
			if r == '/' && (nextR == '/' || nextR == '*') {
				return true
			}
		}
		prevR = r
	}
	return false
}

// See https://sourcegraph.com/github.com/elastic/kibana/-/blob/src/platform/plugins/shared/console/public/application/containers/editor/utils/requests_utils.ts?L204.
func getEditorRequest(text string, a adjustedParsedRequest) *editorRequest {
	e := &editorRequest{}
	lines := strings.Split(text, "\n")
	methodURLLine := strings.TrimSpace(lines[a.startLineNumber])
	if methodURLLine == "" {
		return nil
	}
	method, url := parseLine(methodURLLine)
	if method == "" || url == "" {
		return nil
	}
	e.method = method
	e.url = url

	if a.endLineNumber <= a.startLineNumber {
		return e
	}

	dataString := ""
	if a.startLineNumber < len(lines)-1 {
		var validLines []string
		for i := a.startLineNumber + 1; i <= a.endLineNumber; i++ {
			validLines = append(validLines, lines[i])
		}
		dataString = strings.TrimSpace(strings.Join(validLines, "\n"))
	}

	data := splitDataIntoJSONObjects(dataString)
	return &editorRequest{
		method: method,
		url:    url,
		data:   data,
	}
}

// Splits a concatenated string of JSON objects into individual JSON objects.
// This function takes a string containing one or more JSON objects concatenated together,
// separated by optional whitespace, and splits them into an array of individual JSON strings.
// It ensures that nested objects and strings containing braces do not interfere with the splitting logic.
//
// Example inputs:
// - '{ "query": "test"} { "query": "test" }' -> ['{ "query": "test"}', '{ "query": "test" }']
// - '{ "query": "test"}' -> ['{ "query": "test"}']
// - '{ "query": "{a} {b}"}' -> ['{ "query": "{a} {b}"}'].
func splitDataIntoJSONObjects(s string) []string {
	var jsonObjects []string
	// Track the depth of nested braces
	depth := 0
	// Holds the current JSON object as we iterate
	currentObject := ""
	// Tracks whether the current position is inside a string
	insideString := false
	// Iterate through each character in the input string
	rs := []rune(s)
	for i, r := range rs {
		// Append the character to the current JSON object string
		currentObject += string(r)

		// If the character is a double quote and it is not escaped, toggle the `insideString` state
		if r == '"' && (i == 0 || rs[i-1] != '\\') {
			insideString = !insideString
		} else if !insideString {
			// Only modify depth if not inside a string
			switch r {
			case '{':
				depth++
			case '}':
				depth--
			default:
				// Other characters don't affect depth
			}

			if depth == 0 {
				jsonObjects = append(jsonObjects, strings.TrimSpace(currentObject))
				currentObject = ""
			}
		}
	}

	// If there's remaining data in currentObject, add it as the last JSON object.
	if strings.TrimSpace(currentObject) != "" {
		jsonObjects = append(jsonObjects, strings.TrimSpace(currentObject))
	}

	// Filter out any empty strings from the result
	var result []string
	for i := 0; i < len(jsonObjects); i++ {
		if jsonObjects[i] != "" {
			result = append(result, jsonObjects[i])
		}
	}
	return result
}

func parseLine(line string) (method string, url string) {
	line = strings.TrimSpace(line)
	firstWhitespaceIndex := strings.Index(line, " ")
	if firstWhitespaceIndex < 0 {
		// There is no url, only method
		return line, ""
	}

	// 1st part is the method
	method = strings.ToUpper(strings.TrimSpace(line[0:firstWhitespaceIndex]))
	// 2nd part is the url
	url = removeTrailingWhitespace(strings.TrimSpace(line[firstWhitespaceIndex:]))
	return method, url
}

// This function removes any trailing comments, for example:
// "_search // comment" -> "_search"
// Ideally the parser would do that, but currently they are included in the url.
func removeTrailingWhitespace(s string) string {
	index := 0
	whitespaceIndex := -1
	isQueryParam := false
	for {
		r, sz := utf8.DecodeRuneInString(s[index:])
		if r == utf8.RuneError {
			break
		}
		if r == '"' {
			isQueryParam = !isQueryParam
		} else if r == ' ' && !isQueryParam {
			whitespaceIndex = index
			break
		}
		index += sz
	}
	if whitespaceIndex > 0 {
		return s[:whitespaceIndex]
	}
	return s
}

func newParser(text string) *parser {
	return &parser{
		at: 0,
		ch: 0,
		escapee: map[rune]string{
			'"':  `"`,
			'\\': `\`,
			'/':  `/`,
			'b':  "\b",
			'f':  "\f",
			'n':  "\n",
			'r':  "\r",
			't':  "\t",
		},
		text:   text,
		errors: []*syntaxError{},
	}
}

// See https://sourcegraph.com/github.com/elastic/kibana/-/blob/src/platform/packages/shared/kbn-monaco/src/languages/console/parser.js.
func (p *parser) parse() ([]parsedRequest, error) {
	if _, err := p.nextEmptyInput(); err != nil {
		return nil, err
	}
	if err := p.multiRequest(); err != nil {
		return nil, err
	}

	if err := p.white(); err != nil {
		return nil, err
	}
	if p.ch != 0 {
		return nil, errors.Errorf("Syntax error")
	}
	return p.requests, nil
}

func (p *parser) multiRequest() error {
	catch := func(e error) int {
		p.errors = append(p.errors, &syntaxError{
			byteOffset: p.at,
			message:    e.Error(),
		})
		if p.at >= len(p.text) {
			return -1
		}
		remain := p.text[p.at:]
		re := regexp.MustCompile(`(?m)^(POST|HEAD|GET|PUT|DELETE|PATCH)`)
		match := re.FindStringIndex(remain)
		if match != nil {
			return match[0] + p.at
		}
		return -1
	}
	for p.ch != 0 {
		if err := p.white(); err != nil {
			if next := catch(err); next >= 0 {
				if err := p.reset(next); err != nil {
					return err
				}
			} else {
				return nil
			}
		}
		if p.ch == 0 {
			continue
		}
		if err := p.comment(); err != nil {
			if next := catch(err); next >= 0 {
				if err := p.reset(next); err != nil {
					return err
				}
			} else {
				return nil
			}
		}
		if err := p.white(); err != nil {
			if next := catch(err); next >= 0 {
				if err := p.reset(next); err != nil {
					return err
				}
			} else {
				return nil
			}
		}
		if p.ch == 0 {
			continue
		}
		if err := p.request(); err != nil {
			if next := catch(err); next >= 0 {
				if err := p.reset(next); err != nil {
					return err
				}
			} else {
				return nil
			}
		}
		if err := p.white(); err != nil {
			if next := catch(err); next >= 0 {
				if err := p.reset(next); err != nil {
					return err
				}
			} else {
				return nil
			}
		}
	}

	return nil
}

func (p *parser) request() error {
	if err := p.white(); err != nil {
		return err
	}
	if err := p.addRequestStart(); err != nil {
		return err
	}
	if _, err := p.method(); err != nil {
		return err
	}
	if err := p.updateRequestEnd(); err != nil {
		return err
	}
	if err := p.strictWhite(); err != nil {
		return err
	}
	if _, err := p.url(); err != nil {
		return err
	}
	if err := p.updateRequestEnd(); err != nil {
		return err
	}
	// advance to one new line
	if err := p.strictWhite(); err != nil {
		return err
	}
	if err := p.newLine(); err != nil {
		return err
	}
	if err := p.strictWhite(); err != nil {
		return err
	}
	if p.ch == '{' {
		if _, err := p.object(); err != nil {
			return err
		}
		if err := p.updateRequestEnd(); err != nil {
			return err
		}
	}
	// multi doc request
	// advance to one new line
	if err := p.strictWhite(); err != nil {
		return err
	}
	if err := p.newLine(); err != nil {
		return err
	}
	if err := p.strictWhite(); err != nil {
		return err
	}
	for p.ch == '{' {
		// another object
		if _, err := p.object(); err != nil {
			return err
		}
		if err := p.updateRequestEnd(); err != nil {
			return err
		}
		if err := p.strictWhite(); err != nil {
			return err
		}
		if err := p.newLine(); err != nil {
			return err
		}
		if err := p.strictWhite(); err != nil {
			return err
		}
	}
	return p.addRequestEnd()
}

func (p *parser) url() (string, error) {
	url := ""
	for p.ch != 0 && p.ch != '\n' {
		url += string(p.ch)
		if _, err := p.nextEmptyInput(); err != nil {
			return "", err
		}
	}
	if url == "" {
		return "", errors.Errorf("Missing url")
	}
	return url, nil
}

func (p *parser) object() (map[string]any, error) {
	key := ""
	object := make(map[string]any)

	if p.ch == '{' {
		if err := p.next('{'); err != nil {
			return nil, err
		}
		if err := p.white(); err != nil {
			return nil, err
		}
		if p.ch == '}' {
			if err := p.next('}'); err != nil {
				return nil, err
			}
			// empty object
			return object, nil
		}
		for p.ch != 0 {
			var err error
			key, err = p.string()
			if err != nil {
				return nil, err
			}
			if err := p.white(); err != nil {
				return nil, err
			}
			if err := p.next(':'); err != nil {
				return nil, err
			}
			if _, ok := object[key]; ok {
				return nil, errors.Errorf("duplicate key '%s'", key)
			}
			v, err := p.value()
			if err != nil {
				return nil, err
			}
			object[key] = v
			if err := p.white(); err != nil {
				return nil, err
			}
			if p.ch == '}' {
				if err := p.next('}'); err != nil {
					return nil, err
				}
				return object, nil
			}
			if err := p.next(','); err != nil {
				return nil, err
			}
			if err := p.white(); err != nil {
				return nil, err
			}
		}
	}
	return nil, errors.Errorf("bad object")
}

func (p *parser) value() (any, error) {
	if err := p.white(); err != nil {
		return nil, err
	}
	switch p.ch {
	case '{':
		return p.object()
	case '[':
		return p.array()
	case '"':
		return p.string()
	case '-':
		return p.number()
	default:
		if p.ch >= '0' && p.ch <= '9' {
			return p.number()
		}
		return p.word()
	}
}

func (p *parser) word() (any, error) {
	switch p.ch {
	case 't':
		if err := p.next('t'); err != nil {
			return nil, err
		}
		if err := p.next('r'); err != nil {
			return nil, err
		}
		if err := p.next('u'); err != nil {
			return nil, err
		}
		if err := p.next('e'); err != nil {
			return nil, err
		}
		return true, nil
	case 'f':
		if err := p.next('f'); err != nil {
			return nil, err
		}
		if err := p.next('a'); err != nil {
			return nil, err
		}
		if err := p.next('l'); err != nil {
			return nil, err
		}
		if err := p.next('s'); err != nil {
			return nil, err
		}
		if err := p.next('e'); err != nil {
			return nil, err
		}
		return false, nil
	case 'n':
		if err := p.next('n'); err != nil {
			return nil, err
		}
		if err := p.next('u'); err != nil {
			return nil, err
		}
		if err := p.next('l'); err != nil {
			return nil, err
		}
		if err := p.next('l'); err != nil {
			return nil, err
		}
		return nil, nil
	default:
		return nil, errors.Errorf("unexpected '%c'", p.ch)
	}
}

func (p *parser) number() (float64, error) {
	s := ""
	if p.ch == '-' {
		s = "-"
		if err := p.next('-'); err != nil {
			return 0, err
		}
	}
	for p.ch >= '0' && p.ch <= '9' {
		s += string(p.ch)
		if _, err := p.nextEmptyInput(); err != nil {
			return 0, err
		}
	}
	if p.ch == '.' {
		s += "."
		for {
			if _, err := p.nextEmptyInput(); err != nil {
				return 0, err
			}
			if p.ch >= '0' && p.ch <= '9' {
				s += string(p.ch)
			} else {
				break
			}
		}
	}
	if p.ch == 'e' || p.ch == 'E' {
		s += string(p.ch)
		if _, err := p.nextEmptyInput(); err != nil {
			return 0, err
		}
		if p.ch == '+' || p.ch == '-' {
			s += string(p.ch)
			if _, err := p.nextEmptyInput(); err != nil {
				return 0, err
			}
		}
		for p.ch >= '0' && p.ch <= '9' {
			s += string(p.ch)
			if _, err := p.nextEmptyInput(); err != nil {
				return 0, err
			}
		}
	}
	num, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errors.Errorf("bad number")
	}
	return num, nil
}

func (p *parser) array() ([]any, error) {
	var array []any
	if p.ch == '[' {
		if err := p.next('['); err != nil {
			return nil, err
		}
		if err := p.white(); err != nil {
			return nil, err
		}
		if p.ch == ']' {
			if err := p.next(']'); err != nil {
				return nil, err
			}
			// empty array
			return array, nil
		}
		for p.ch != 0 {
			var err error
			v, err := p.value()
			if err != nil {
				return nil, err
			}
			array = append(array, v)
			if err := p.white(); err != nil {
				return nil, err
			}
			if p.ch == ']' {
				if err := p.next(']'); err != nil {
					return nil, err
				}
				return array, nil
			}
			if err := p.next(','); err != nil {
				return nil, err
			}
			if err := p.white(); err != nil {
				return nil, err
			}
		}
	}
	return nil, errors.Errorf("bad array")
}

func (p *parser) string() (string, error) {
	s := ""
	uffff := 0
	if p.ch == '"' {
		if p.peek(0) == '"' && p.peek(1) == '"' {
			// literal
			if err := p.next('"'); err != nil {
				return "", err
			}
			if err := p.next('"'); err != nil {
				return "", err
			}
			return p.nextUpTo(`"""`, `failed to find closing '"""'`)
		}
		for {
			r, err := p.nextEmptyInput()
			if err != nil {
				return "", err
			}
			if r == 0 {
				break
			}
			if p.ch == '"' {
				if _, err := p.nextEmptyInput(); err != nil {
					return "", err
				}
				return s, nil
			} else if p.ch == '\\' {
				if _, err := p.nextEmptyInput(); err != nil {
					return "", err
				}
				if p.ch == 'u' {
					uffff = 0
					for i := 0; i < 4; i++ {
						nextRune, err := p.nextEmptyInput()
						if err != nil {
							return "", err
						}
						// Parse next rune into hex.
						hex, err := strconv.ParseUint(string(nextRune), 16, 32)
						if err != nil {
							break
						}
						uffff = (uffff << 4) + int(hex)
					}
					// Treat uffff as UTF-16 encoded rune.
					s += string(rune(uffff))
				} else if v, ok := p.escapee[p.ch]; ok {
					s += v
				} else {
					break
				}
			} else {
				s += string(p.ch)
			}
		}
	}

	return "", errors.Errorf("bad string")
}

func (p *parser) nextUpTo(upTo string, errorMessage string) (string, error) {
	currentAt := p.at
	i := strings.Index(p.text[p.at:], upTo)
	if i < 0 {
		if errorMessage != "" {
			return "", errors.New(errorMessage)
		}
		return "", errors.Errorf("expected '%s'", upTo)
	}
	i += currentAt
	if err := p.reset(i + len(upTo)); err != nil {
		return "", err
	}
	return p.text[currentAt:i], nil
}

func (p *parser) reset(newAt int) error {
	ch, sz := utf8.DecodeRuneInString(p.text[newAt:])
	if ch == utf8.RuneError {
		if sz == 0 {
			return errors.Errorf("unexpected empty input")
		}
		if sz == 1 {
			return errors.Errorf("invalid UTF-8 character")
		}
		return errors.Errorf("unknown decoding rune error")
	}
	p.ch = ch
	p.at = newAt + sz
	return nil
}

func (p *parser) newLine() error {
	if p.ch == '\n' {
		if _, err := p.nextEmptyInput(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) strictWhite() error {
	for p.ch != 0 && (p.ch == ' ' || p.ch == '\t') {
		if _, err := p.nextEmptyInput(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) nextOneOf(rs []rune) error {
	if !includes(rs, p.ch) {
		return errors.Errorf("expected one of %+v instead of '%c'", rs, p.ch)
	}
	if p.at >= len(p.text) {
		// EOF, just increase the at by 1 and set the ch to 0.
		p.at++
		p.ch = 0
		return nil
	}
	ch, sz := utf8.DecodeRuneInString(p.text[p.at:])
	if ch == utf8.RuneError {
		if sz == 0 {
			return errors.Errorf("unexpected empty input")
		}
		if sz == 1 {
			return errors.Errorf("invalid UTF-8 character")
		}
		return errors.Errorf("unknown decoding rune error")
	}
	p.ch = ch
	p.at += sz
	return nil
}

func (p *parser) method() (string, error) {
	uppercase := strings.ToUpper(string(p.ch))
	switch uppercase {
	case "G":
		if err := p.nextOneOf([]rune{'G', 'g'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'E', 'e'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'T', 't'}); err != nil {
			return "", err
		}
		return "GET", nil
	case "H":
		if err := p.nextOneOf([]rune{'H', 'h'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'E', 'e'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'A', 'a'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'D', 'd'}); err != nil {
			return "", err
		}
		return "HEAD", nil
	case "D":
		if err := p.nextOneOf([]rune{'D', 'd'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'E', 'e'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'L', 'l'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'E', 'e'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'T', 't'}); err != nil {
			return "", err
		}
		if err := p.nextOneOf([]rune{'E', 'e'}); err != nil {
			return "", err
		}
		return "DELETE", nil
	case "P":
		if err := p.nextOneOf([]rune{'P', 'p'}); err != nil {
			return "", err
		}
		nextUppercase := strings.ToUpper(string(p.ch))
		switch nextUppercase {
		case "A":
			if err := p.nextOneOf([]rune{'A', 'a'}); err != nil {
				return "", err
			}
			if err := p.nextOneOf([]rune{'T', 't'}); err != nil {
				return "", err
			}
			if err := p.nextOneOf([]rune{'C', 'c'}); err != nil {
				return "", err
			}
			if err := p.nextOneOf([]rune{'H', 'h'}); err != nil {
				return "", err
			}
			return "PATCH", nil
		case "U":
			if err := p.nextOneOf([]rune{'U', 'u'}); err != nil {
				return "", err
			}
			if err := p.nextOneOf([]rune{'T', 't'}); err != nil {
				return "", err
			}
			return "PUT", nil
		case "O":
			if err := p.nextOneOf([]rune{'O', 'o'}); err != nil {
				return "", err
			}
			if err := p.nextOneOf([]rune{'S', 's'}); err != nil {
				return "", err
			}
			if err := p.nextOneOf([]rune{'T', 't'}); err != nil {
				return "", err
			}
			return "POST", nil
		default:
			return "", errors.Errorf("unexpected '%c'", p.ch)
		}
	default:
		return "", errors.Errorf("expected one of GET/POST/PUT/DELETE/HEAD/PATCH")
	}
}

func (p *parser) addRequestStart() error {
	previousRune, sz := utf8.DecodeLastRuneInString(p.text[:p.at])
	if previousRune == utf8.RuneError {
		if sz == 0 {
			return errors.Errorf("unexpected empty input")
		}
		if sz == 1 {
			return errors.Errorf("invalid UTF-8 character")
		}
		return errors.Errorf("unknown decoding rune error")
	}
	p.requestStartOffset = p.at - sz
	p.requests = append(p.requests, parsedRequest{
		startOffset: p.requestStartOffset,
		endOffset:   -1,
	})
	return nil
}

func (p *parser) addRequestEnd() error {
	if len(p.requests) == 0 {
		return errors.Errorf("unexpected empty requests")
	}
	p.requests[len(p.requests)-1].endOffset = p.requestEndOffset
	return nil
}

func (p *parser) updateRequestEnd() error {
	if p.at >= len(p.text) {
		p.requestEndOffset = p.at - 1
		return nil
	}
	previousRune, sz := utf8.DecodeLastRuneInString(p.text[:p.at])
	if previousRune == utf8.RuneError {
		if sz == 0 {
			return errors.Errorf("unexpected empty input")
		}
		if sz == 1 {
			return errors.Errorf("invalid UTF-8 character")
		}
		return errors.Errorf("unknown decoding rune error")
	}
	p.requestEndOffset = p.at - sz
	return nil
}

func (p *parser) comment() error {
	for p.ch == '#' {
		for p.ch != 0 && p.ch != '\n' {
			if _, err := p.nextEmptyInput(); err != nil {
				return err
			}
		}
		if err := p.white(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) peek(offset uint) rune {
	if p.at >= len(p.text) {
		return 0
	}
	tempAt := p.at
	var peekCh rune
	var sz int
	for i := uint(0); i <= offset; i++ {
		if tempAt >= len(p.text) {
			return 0
		}
		peekCh, sz = utf8.DecodeRuneInString(p.text[tempAt:])
		if peekCh == utf8.RuneError {
			return 0
		}
		tempAt += sz
	}
	return peekCh
}

func (p *parser) white() error {
	for p.ch != 0 {
		// Skip whitespace.
		for p.ch != 0 && p.ch <= ' ' {
			if _, err := p.nextEmptyInput(); err != nil {
				return err
			}
		}

		// if the current rune in iteration is '#' or the rune and the next rune is equal to '//'
		// we are on the single line comment.
		if p.ch == '#' || (p.ch == '/' && p.peek(0) == '/') {
			// Until we are on the new line, skip to the next char.
			for p.ch != 0 && p.ch != '\n' {
				if _, err := p.nextEmptyInput(); err != nil {
					return err
				}
			}
		} else if p.ch == '/' && p.peek(0) == '*' {
			// If the chars starts with '/*', we are on the multiline comment.
			if err := p.nNextEmptyInput(2); err != nil {
				return err
			}
			for p.ch != 0 && (p.ch != '*' || p.peek(0) != '/') {
				// Until we have closing tags '*', skip to the next char.
				if _, err := p.nextEmptyInput(); err != nil {
					return err
				}
			}
			if p.ch != 0 {
				if err := p.nNextEmptyInput(2); err != nil {
					return err
				}
			}
		} else {
			break
		}
	}

	return nil
}

func (p *parser) next(c rune) error {
	if c != p.ch {
		return errors.Errorf("expected '%c' instead of '%c'", c, p.ch)
	}

	_, err := p.nextEmptyInput()
	return err
}

func (p *parser) nextEmptyInput() (rune, error) {
	if p.at >= len(p.text) {
		// EOF
		p.at++
		p.ch = 0
		return 0, nil
	}
	nextCh, sz := utf8.DecodeRuneInString(p.text[p.at:])
	if nextCh == utf8.RuneError {
		if sz == 0 {
			return 0, errors.Errorf("unexpected empty input")
		}
		if sz == 1 {
			return 0, errors.Errorf("invalid UTF-8 character")
		}
		return 0, errors.Errorf("unknown decoding rune error")
	}
	p.ch = nextCh

	p.at += sz
	return nextCh, nil
}

// call p.nextEmptyInput n times.
func (p *parser) nNextEmptyInput(n int) error {
	var err error
	for i := 0; i < n; i++ {
		if _, err = p.nextEmptyInput(); err != nil {
			return err
		}
	}
	return nil
}

func includes[T rune](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
