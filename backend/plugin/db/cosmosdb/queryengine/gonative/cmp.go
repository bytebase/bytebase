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
// That is the order the service itself returns mixed types in, and the class
// ranking is reproducible. The order *within* the array and object classes is
// not: it is the index's own encoding, and a real container answers
// [1,2] < [2] < [1,3] and {"b":1} < {"a":2} < {"a":1}, which no function of the
// values gives back.
//
// So two unequal values of one of those classes have no order here, and the
// two comparators below part ways over them: compareItems fails, because an
// ORDER BY merge that disagrees with the per-partition streams it merges puts
// out rows in an order that is neither the service's nor its own, and under a
// TOP it returns the wrong rows outright. compareItemsTotal orders them by
// canonical JSON, for the one caller that only needs *some* order it can
// repeat — a GROUP BY the user did not order.
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
		order, err := compareComposites(a, b)
		if err != nil {
			return 0, err
		}
		if order == 0 {
			return 0, nil
		}
		// Equal composites are one value and may be ordered together; unequal
		// ones the service separates by an encoding this cannot see.
		return 0, errors.Wrapf(ErrUnsupportedPlanFeature,
			"gonative: ORDER BY cannot order two different %ss", ta)
	default:
		return 0, errors.Errorf("gonative: unknown jsonType %d", ta)
	}
}

// compareItemsTotal is compareItems extended to a total order: two unequal
// values of one composite class are ordered by their canonical JSON. Only a
// caller free to pick the order may use it — see jsonType — because that
// order is not the service's.
func compareItemsTotal(a, b json.RawMessage) (int, error) {
	ta, tb := detectJSONType(a), detectJSONType(b)
	if ta == tb && (ta == jsonArray || ta == jsonObject) {
		return compareComposites(a, b)
	}
	return compareItems(a, b)
}

// compareComposites orders two arrays or two objects by their canonical JSON,
// which is 0 exactly when they are the same value.
func compareComposites(a, b json.RawMessage) (int, error) {
	ca, err := canonicalJSON(a)
	if err != nil {
		return 0, errors.Wrap(err, "gonative: cmp composite a")
	}
	cb, err := canonicalJSON(b)
	if err != nil {
		return 0, errors.Wrap(err, "gonative: cmp composite b")
	}
	return bytes.Compare(ca, cb), nil
}

// String names the class in an error a user may read.
func (t jsonType) String() string {
	switch t {
	case jsonUndefined:
		return "undefined"
	case jsonNull:
		return "null"
	case jsonBool:
		return "boolean"
	case jsonNumber:
		return "number"
	case jsonString:
		return "string"
	case jsonArray:
		return "array"
	case jsonObject:
		return "object"
	default:
		return "value"
	}
}
