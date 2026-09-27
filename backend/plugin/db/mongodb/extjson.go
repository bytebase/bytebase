package mongodb

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// The MongoDB Go driver formats relaxed Extended JSON doubles with Go's %G
// verb, which switches to scientific notation at 1e6 (e.g.
// 1.779696815227E+12) — notation no MongoDB tool uses.
// normalizeExtJSONNumbers rewrites every double token into the notation
// mongosh displays (ECMAScript Number::toString): plain decimal when the
// value's decimal exponent is within [-6, 21), scientific otherwise,
// shortest round-trip digits, and no ".0" suffix on integral doubles.
//
// A number token is a double if and only if it contains '.', 'e', or 'E':
// the driver suffixes integral doubles with ".0", while int32/int64
// serialize as bare digits. Non-finite doubles and Decimal128 arrive as
// {"$numberDouble": "..."} / {"$numberDecimal": "..."} strings, which a
// number-token rewrite never touches. On any parse anomaly the input is
// returned unchanged.
func normalizeExtJSONNumbers(data []byte) []byte {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var out bytes.Buffer
	out.Grow(len(data))
	// Container stack: true for object, false for array.
	var stack []bool
	needComma := false
	expectKey := false

	completeValue := func() {
		needComma = true
		expectKey = len(stack) > 0 && stack[len(stack)-1]
	}

	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return data
		}

		if delim, ok := token.(json.Delim); ok && (delim == '}' || delim == ']') {
			out.WriteByte(byte(delim))
			stack = stack[:len(stack)-1]
			completeValue()
			continue
		}

		if needComma {
			out.WriteByte(',')
			needComma = false
		}

		switch v := token.(type) {
		case json.Delim:
			// '{' or '['; closing delims are handled above.
			out.WriteByte(byte(v))
			stack = append(stack, v == '{')
			expectKey = v == '{'
		case string:
			if err := writeJSONString(&out, v); err != nil {
				return data
			}
			if expectKey {
				out.WriteByte(':')
				expectKey = false
			} else {
				completeValue()
			}
		case json.Number:
			raw := v.String()
			if strings.ContainsAny(raw, ".eE") {
				if f, err := strconv.ParseFloat(raw, 64); err == nil {
					raw = formatDoubleJS(f)
				}
			}
			out.WriteString(raw)
			completeValue()
		case bool:
			out.WriteString(strconv.FormatBool(v))
			completeValue()
		case nil:
			out.WriteString("null")
			completeValue()
		default:
			return data
		}
	}

	if len(stack) != 0 {
		return data
	}
	return out.Bytes()
}

// writeJSONString encodes s as a JSON string without HTML escaping, matching
// how the driver writes strings ("<" stays "<", not "\u003c").
func writeJSONString(buf *bytes.Buffer, s string) error {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return err
	}
	// Encode appends a newline; drop it.
	buf.Truncate(buf.Len() - 1)
	return nil
}

// formatDoubleJS renders a finite float64 the way ECMAScript
// Number::toString does — and therefore the way mongosh, Compass, and
// DataGrip display it. The one deviation is negative zero, rendered "-0" to
// keep the sign actually stored in the BSON double (String(-0) is "0" in
// ECMAScript).
func formatDoubleJS(f float64) string {
	// Shortest round-trip digits in the form d[.ddd]e±XX.
	s := strconv.FormatFloat(f, 'e', -1, 64)
	mantissa, expStr, found := strings.Cut(s, "e")
	if !found {
		// ±Inf/NaN; unreachable from JSON number tokens.
		return s
	}
	exp, err := strconv.Atoi(expStr)
	if err != nil {
		return s
	}
	sign := ""
	if strings.HasPrefix(mantissa, "-") {
		sign = "-"
		mantissa = mantissa[1:]
	}
	digits := strings.Replace(mantissa, ".", "", 1)
	// ECMAScript Number::toString(10): the value is digits × 10^(n-k) with
	// k significant digits; plain decimal notation applies for -6 < n <= 21.
	n := exp + 1
	k := len(digits)
	switch {
	case k <= n && n <= 21:
		return sign + digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return sign + digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return sign + "0." + strings.Repeat("0", -n) + digits
	default:
		e := n - 1
		expPart := "e+"
		if e < 0 {
			expPart = "e-"
			e = -e
		}
		if k == 1 {
			return sign + digits + expPart + strconv.Itoa(e)
		}
		return sign + digits[:1] + "." + digits[1:] + expPart + strconv.Itoa(e)
	}
}
