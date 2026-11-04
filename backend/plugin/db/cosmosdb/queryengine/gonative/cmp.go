package gonative

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

// jsonType enumerates the Cosmos item-ordering classes in ascending order:
//
//	undefined < null < bool < number < string < array < object
//
// That is the order the service itself returns mixed types in. Within the
// array and object classes its order is the index's own encoding — a real
// container answers [1,2] < [2] < [1,3] and {"b":1} < {"a":2} < {"a":1} — so
// it is not a value order and cannot be reproduced from the values alone.
// Two values of one class are therefore ordered by their canonical JSON,
// which is stable across runs and leaves the class ranking above, the part a
// user can predict, matching the service.
type jsonType int

const (
	jsonUndefined jsonType = iota
	jsonNull
	jsonBool
	jsonNumber
	jsonString
	jsonArray
	jsonObject
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
	case '[':
		return jsonArray
	case '{':
		return jsonObject
	}
	// digit, minus, plus, or dot → number
	if (trimmed[0] >= '0' && trimmed[0] <= '9') || trimmed[0] == '-' || trimmed[0] == '+' || trimmed[0] == '.' {
		return jsonNumber
	}
	return jsonUndefined
}

// compareItems compares two JSON-encoded values using Cosmos item ordering.
// Return values follow the usual convention: -1, 0, +1. Every shape a document
// can hold compares — see jsonType for how arrays and objects are ranked — so
// an error here means the bytes were not JSON.
func compareItems(a, b json.RawMessage) (int, error) {
	ta, tb := detectJSONType(a), detectJSONType(b)
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
	case jsonArray, jsonObject:
		ca, err := canonicalJSON(a)
		if err != nil {
			return 0, errors.Wrap(err, "gonative: cmp composite a")
		}
		cb, err := canonicalJSON(b)
		if err != nil {
			return 0, errors.Wrap(err, "gonative: cmp composite b")
		}
		return bytes.Compare(ca, cb), nil
	default:
		return 0, errors.Errorf("gonative: unknown jsonType %d", ta)
	}
}
