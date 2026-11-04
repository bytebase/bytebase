package gonative

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

// jsonType enumerates the Cosmos item-ordering classes in ascending order:
//
//	undefined < null < bool < number < string
//
// Arrays and objects are not orderable in the Cosmos item-comparison rules;
// callers that encounter them should treat the result as an error.
type jsonType int

const (
	jsonUndefined jsonType = iota
	jsonNull
	jsonBool
	jsonNumber
	jsonString
	jsonUnorderable // arrays, objects — reject rather than compare
)

// detectJSONType inspects raw JSON (possibly leading whitespace) and returns
// its Cosmos ordering class.
func detectJSONType(raw json.RawMessage) jsonType {
	trimmed := bytes.TrimLeft(raw, " \t\n\r")
	if len(trimmed) == 0 {
		return jsonUndefined
	}
	switch trimmed[0] {
	case 'n':
		return jsonNull
	case 't', 'f':
		return jsonBool
	case '"':
		return jsonString
	case '[', '{':
		return jsonUnorderable
	}
	// digit, minus, plus, or dot → number
	if (trimmed[0] >= '0' && trimmed[0] <= '9') || trimmed[0] == '-' || trimmed[0] == '+' || trimmed[0] == '.' {
		return jsonNumber
	}
	return jsonUndefined
}

// compareItems compares two JSON-encoded scalars using Cosmos item ordering.
// Return values follow the usual convention: -1, 0, +1.
// Returns an error when either side is unorderable (array/object).
func compareItems(a, b json.RawMessage) (int, error) {
	ta, tb := detectJSONType(a), detectJSONType(b)
	if ta == jsonUnorderable || tb == jsonUnorderable {
		return 0, errors.New("gonative: cannot compare unorderable values (arrays/objects)")
	}
	if ta != tb {
		if ta < tb {
			return -1, nil
		}
		return 1, nil
	}
	// Same type — compare within the type.
	switch ta {
	case jsonUndefined, jsonNull:
		// All undefined == undefined; all null == null.
		return 0, nil
	case jsonBool:
		var av, bv bool
		if err := json.Unmarshal(a, &av); err != nil {
			return 0, errors.Wrap(err, "gonative: cmp bool a")
		}
		if err := json.Unmarshal(b, &bv); err != nil {
			return 0, errors.Wrap(err, "gonative: cmp bool b")
		}
		switch {
		case av == bv:
			return 0, nil
		case !av && bv:
			return -1, nil
		default:
			return 1, nil
		}
	case jsonNumber:
		var av, bv float64
		if err := json.Unmarshal(a, &av); err != nil {
			return 0, errors.Wrap(err, "gonative: cmp number a")
		}
		if err := json.Unmarshal(b, &bv); err != nil {
			return 0, errors.Wrap(err, "gonative: cmp number b")
		}
		switch {
		case av == bv:
			return 0, nil
		case av < bv:
			return -1, nil
		default:
			return 1, nil
		}
	case jsonString:
		var av, bv string
		if err := json.Unmarshal(a, &av); err != nil {
			return 0, errors.Wrap(err, "gonative: cmp string a")
		}
		if err := json.Unmarshal(b, &bv); err != nil {
			return 0, errors.Wrap(err, "gonative: cmp string b")
		}
		return strings.Compare(av, bv), nil
	default:
		return 0, errors.Errorf("gonative: unknown jsonType %d", ta)
	}
}
