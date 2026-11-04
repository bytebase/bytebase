package gonative

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
)

// aggregator is the interface the pipeline uses to accumulate per-partition
// partial values and produce a single finalized scalar.
//
// feedPartial receives a JSON-encoded partial whose shape depends on the
// aggregate kind:
//
//	Count, Sum        — bare number, e.g. 18 or 20442258
//	Avg               — {"sum": <number>, "count": <int>}
//	Min               — {"min": <value>, "count": <int>}
//	Max               — {"max": <value>, "count": <int>}
//
// finalize returns the final scalar as a json.RawMessage so the pipeline can
// emit it without re-marshaling. Returns an empty value (JSON null) when no
// contributing rows were observed (e.g. Avg over an empty set).
type aggregator interface {
	feedPartial(raw json.RawMessage) error
	finalize() (json.RawMessage, error)
}

// newAggregator constructs an aggregator for the named kind. Returns nil for
// an unrecognized kind so the caller can surface an explicit error.
//
// Gateway plans emit either "Average" or "Avg" for the mean aggregate.
// Accepting both avoids surprises if a future gateway version shifts.
func newAggregator(kind string) aggregator {
	switch kind {
	case "Count":
		return &countAgg{}
	case "Sum":
		return &sumAgg{}
	case "Min":
		return &minAgg{}
	case "Max":
		return &maxAgg{}
	case "Average", "Avg":
		return &avgAgg{}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Count

type countAgg struct {
	total float64
}

func (a *countAgg) feedPartial(raw json.RawMessage) error {
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil {
		return errors.Wrap(err, "gonative/agg.count")
	}
	a.total += n
	return nil
}

func (a *countAgg) finalize() (json.RawMessage, error) {
	return numberRawMessage(a.total), nil
}

// -----------------------------------------------------------------------------
// Sum

type sumAgg struct {
	total float64
	seen  bool
}

func (a *sumAgg) feedPartial(raw json.RawMessage) error {
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil {
		return errors.Wrap(err, "gonative/agg.sum")
	}
	a.total += n
	a.seen = true
	return nil
}

func (a *sumAgg) finalize() (json.RawMessage, error) {
	if !a.seen {
		return json.RawMessage(`null`), nil
	}
	return numberRawMessage(a.total), nil
}

// -----------------------------------------------------------------------------
// Min

type minAgg struct {
	best json.RawMessage
}

type minPartial struct {
	Min   json.RawMessage `json:"min"`
	Count int             `json:"count"`
}

func (a *minAgg) feedPartial(raw json.RawMessage) error {
	var p minPartial
	if err := json.Unmarshal(raw, &p); err != nil {
		return errors.Wrap(err, "gonative/agg.min")
	}
	if p.Count == 0 {
		// Partition contributed no values — ignore.
		return nil
	}
	if a.best == nil {
		a.best = append([]byte(nil), p.Min...)
		return nil
	}
	cmp, err := compareItems(p.Min, a.best)
	if err != nil {
		return errors.Wrap(err, "gonative/agg.min")
	}
	if cmp < 0 {
		a.best = append([]byte(nil), p.Min...)
	}
	return nil
}

func (a *minAgg) finalize() (json.RawMessage, error) {
	if a.best == nil {
		return json.RawMessage(`null`), nil
	}
	return a.best, nil
}

// -----------------------------------------------------------------------------
// Max

type maxAgg struct {
	best json.RawMessage
}

type maxPartial struct {
	Max   json.RawMessage `json:"max"`
	Count int             `json:"count"`
}

func (a *maxAgg) feedPartial(raw json.RawMessage) error {
	var p maxPartial
	if err := json.Unmarshal(raw, &p); err != nil {
		return errors.Wrap(err, "gonative/agg.max")
	}
	if p.Count == 0 {
		return nil
	}
	if a.best == nil {
		a.best = append([]byte(nil), p.Max...)
		return nil
	}
	cmp, err := compareItems(p.Max, a.best)
	if err != nil {
		return errors.Wrap(err, "gonative/agg.max")
	}
	if cmp > 0 {
		a.best = append([]byte(nil), p.Max...)
	}
	return nil
}

func (a *maxAgg) finalize() (json.RawMessage, error) {
	if a.best == nil {
		return json.RawMessage(`null`), nil
	}
	return a.best, nil
}

// -----------------------------------------------------------------------------
// Avg

type avgAgg struct {
	sum   float64
	count int64
}

type avgPartial struct {
	Sum   float64 `json:"sum"`
	Count int64   `json:"count"`
}

func (a *avgAgg) feedPartial(raw json.RawMessage) error {
	var p avgPartial
	if err := json.Unmarshal(raw, &p); err != nil {
		return errors.Wrap(err, "gonative/agg.avg")
	}
	a.sum += p.Sum
	a.count += p.Count
	return nil
}

func (a *avgAgg) finalize() (json.RawMessage, error) {
	if a.count == 0 {
		// Matches Cosmos semantics: AVG over an empty set is undefined;
		// gateway serializes this as JSON null for cross-partition results.
		return json.RawMessage(`null`), nil
	}
	return numberRawMessage(a.sum / float64(a.count)), nil
}

// numberRawMessage encodes a float64 as a compact JSON number. Prefers integer
// formatting when the value is an exact integer — matches the .NET SDK's
// observed behavior of emitting `18` rather than `18.0`.
func numberRawMessage(v float64) json.RawMessage {
	if v == float64(int64(v)) {
		return json.RawMessage(strconv.FormatInt(int64(v), 10))
	}
	return json.RawMessage(strconv.FormatFloat(v, 'g', -1, 64))
}
