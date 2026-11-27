package base

import (
	"errors"

	omnianalysis "github.com/bytebase/omni/analysis"
)

// OmniQueryType maps an omni shared analysis.QueryType onto QueryType.
func OmniQueryType(t omnianalysis.QueryType) QueryType {
	switch t {
	case omnianalysis.Select:
		return Select
	case omnianalysis.Explain:
		return Explain
	case omnianalysis.SelectInfoSchema:
		return SelectInfoSchema
	case omnianalysis.DDL:
		return DDL
	case omnianalysis.DML:
		return DML
	default:
		return QueryTypeUnknown
	}
}

// omniColumnResource converts an omni ColumnResource to a Bytebase ColumnResource.
func omniColumnResource(c omnianalysis.ColumnResource) ColumnResource {
	return ColumnResource{
		Instance: c.Instance,
		Server:   c.Server,
		Database: c.Database,
		Schema:   c.Schema,
		Table:    c.Table,
		Column:   c.Column,
	}
}

// OmniColumnSet converts an omni ColumnSet into SourceColumnSet.
func OmniColumnSet(set omnianalysis.ColumnSet) SourceColumnSet {
	if set == nil {
		return nil
	}
	out := make(SourceColumnSet, len(set))
	for col := range set {
		out[omniColumnResource(col)] = true
	}
	return out
}

// OmniQuerySpanResults converts omni QuerySpanResult entries into []QuerySpanResult.
func OmniQuerySpanResults(results []omnianalysis.QuerySpanResult) []QuerySpanResult {
	if results == nil {
		return nil
	}
	out := make([]QuerySpanResult, 0, len(results))
	for _, r := range results {
		out = append(out, QuerySpanResult{
			Name:           r.Name,
			SourceColumns:  OmniColumnSet(r.SourceColumns),
			IsPlainField:   r.IsPlainField,
			SelectAsterisk: r.SelectAsterisk,
			Invisible:      r.Invisible,
		})
	}
	return out
}

// OmniResourceNotFoundError converts an omni ResourceNotFoundError to Bytebase's ResourceNotFoundError.
func OmniResourceNotFoundError(err *omnianalysis.ResourceNotFoundError) *ResourceNotFoundError {
	if err == nil {
		return nil
	}
	return &ResourceNotFoundError{
		Err:          err.Err,
		Server:       err.Server,
		DatabaseLink: err.DatabaseLink,
		Database:     err.Database,
		Schema:       err.Schema,
		Table:        err.Table,
		Column:       err.Column,
		Function:     err.Function,
	}
}

// omniNotFoundError wraps a lookup error carrying a ResourceNotFoundError so
// errors.As matches *omnianalysis.ResourceNotFoundError while preserving the
// original error's text and unwrap chain.
type omniNotFoundError struct {
	err      error
	notFound *omnianalysis.ResourceNotFoundError
}

func (e omniNotFoundError) Error() string { return e.err.Error() }

func (e omniNotFoundError) Unwrap() error { return e.err }

func (e omniNotFoundError) As(target any) bool {
	if t, ok := target.(**omnianalysis.ResourceNotFoundError); ok {
		*t = e.notFound
		return true
	}
	return false
}

// ToOmniNotFoundError adapts an error carrying a Bytebase ResourceNotFoundError
// so errors.As matches *omnianalysis.ResourceNotFoundError while keeping the
// original error's text and unwrap chain.
func ToOmniNotFoundError(err error) error {
	var notFound *ResourceNotFoundError
	if !errors.As(err, &notFound) {
		return err
	}
	return omniNotFoundError{
		err: err,
		notFound: &omnianalysis.ResourceNotFoundError{
			Err:          notFound.Err,
			Server:       notFound.Server,
			DatabaseLink: notFound.DatabaseLink,
			Database:     notFound.Database,
			Schema:       notFound.Schema,
			Table:        notFound.Table,
			Column:       notFound.Column,
			Function:     notFound.Function,
		},
	}
}

// OmniFunctionNotSupportedError translates an error carrying an omni FunctionNotSupportedError into Bytebase's type.
func OmniFunctionNotSupportedError(err error) error {
	if err == nil {
		return nil
	}
	var unsupported *omnianalysis.FunctionNotSupportedError
	if errors.As(err, &unsupported) {
		return &FunctionNotSupportedError{
			Err:        unsupported.Err,
			Function:   unsupported.Function,
			Definition: unsupported.Definition,
		}
	}
	return err
}

// OmniTypeNotSupportedError converts an omni TypeNotSupportedError to Bytebase's TypeNotSupportedError.
func OmniTypeNotSupportedError(err *omnianalysis.TypeNotSupportedError) *TypeNotSupportedError {
	if err == nil {
		return nil
	}
	return &TypeNotSupportedError{
		Err:   err.Err,
		Type:  err.Type,
		Name:  err.Name,
		Extra: err.Extra,
	}
}

// omniUnresolvedColumnsError converts an omni UnresolvedColumnsError to Bytebase's UnresolvedColumnsError.
func omniUnresolvedColumnsError(err *omnianalysis.UnresolvedColumnsError) *UnresolvedColumnsError {
	if err == nil {
		return nil
	}
	relations := make([]ColumnResource, 0, len(err.Relations))
	for _, relation := range err.Relations {
		relations = append(relations, omniColumnResource(relation))
	}
	return &UnresolvedColumnsError{Relations: relations}
}

// OmniResolvedSpan converts an omni ResolvedSpan into a Bytebase QuerySpan.
func OmniResolvedSpan(span *omnianalysis.ResolvedSpan) *QuerySpan {
	if span == nil {
		return nil
	}
	out := &QuerySpan{
		Type:                   OmniQueryType(span.Type),
		ExplainAnalyze:         span.ExplainAnalyze,
		SourceColumns:          OmniColumnSet(span.SourceColumns),
		Results:                OmniQuerySpanResults(span.Results),
		PredicateColumns:       OmniColumnSet(span.PredicateColumns),
		UnresolvedColumnsError: omniUnresolvedColumnsError(span.UnresolvedColumnsError),
	}
	if span.NotFoundError != nil {
		out.NotFoundError = OmniResourceNotFoundError(span.NotFoundError)
	} else if span.LookupError != nil {
		out.NotFoundError = span.LookupError
	}
	if span.FunctionNotSupportedError != nil {
		out.FunctionNotSupportedError = OmniFunctionNotSupportedError(span.FunctionNotSupportedError)
	}
	return out
}

// ConvertOmniMixTablesError translates omni's ErrMixUserSystemTables into Bytebase's
// MixUserSystemTablesError, preserving wrapping context via mixedTablesError.
// Returns nil if err does not match ErrMixUserSystemTables.
func ConvertOmniMixTablesError(err error) error {
	if err == omnianalysis.ErrMixUserSystemTables {
		return MixUserSystemTablesError
	}
	if errors.Is(err, omnianalysis.ErrMixUserSystemTables) {
		return &mixedTablesError{Cause: err}
	}
	return nil
}

// mixedTablesError wraps an engine mixed-table error so errors.Is matches MixUserSystemTablesError.
type mixedTablesError struct {
	Cause error
}

// Error implements error.
func (e *mixedTablesError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return MixUserSystemTablesError.Error()
}

// Is allows errors.Is to match MixUserSystemTablesError.
func (*mixedTablesError) Is(target error) bool {
	return target == MixUserSystemTablesError
}

// Unwrap returns the underlying engine error.
func (e *mixedTablesError) Unwrap() error {
	return e.Cause
}
