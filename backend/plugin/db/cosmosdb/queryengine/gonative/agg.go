package gonative

import (
	"encoding/json"
	"fmt"
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
// emit it without re-marshaling. It returns a nil message when the aggregate
// has no value: Cosmos reports MIN, MAX and AVG over an empty set as
// undefined, which the gateway renders as no row at all in the VALUE form and
// as an absent alias in the aliased form. COUNT and SUM over an empty set are
// 0, and the gateway sends that 0 as a partial, so they never finalize nil.
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

// aggregatePartial returns the partial feedPartial expects for one aggregate,
// given the gateway's {item, item2} document entry. MIN and MAX read item2:
// only it reports how many rows the partition contributed, which is how a
// partition that matched nothing contributes nothing. item2 is absent where
// the gateway short-circuited to a single-partition shape, and item is then
// the value itself. Every other kind carries its partial in item alone.
//
// Returns nil when the gateway left the partial out, which it does for an
// aggregate over a partition that matched no row.
func aggregatePartial(kind string, it aggItem) json.RawMessage {
	switch kind {
	case "Min", "Max":
		if len(it.Item2) > 0 {
			return it.Item2
		}
		if len(it.Item) == 0 {
			return nil
		}
		key := "min"
		if kind == "Max" {
			key = "max"
		}
		return fmt.Appendf(nil, `{%q:%s,"count":1}`, key, it.Item)
	default:
		return it.Item
	}
}

// feedAggregatePartial folds one partition document's partial into a. A
// partial the gateway left out is not an error: the partition had no row to
// contribute.
func feedAggregatePartial(a aggregator, kind string, it aggItem) error {
	partial := aggregatePartial(kind, it)
	if len(partial) == 0 {
		return nil
	}
	return a.feedPartial(partial)
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
		return nil, nil
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
		return nil, nil
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
		return nil, nil
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
		return nil, nil
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
