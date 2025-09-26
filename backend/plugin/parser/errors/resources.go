package errors

import (
	"fmt"
	"strings"
)

// ResourceNotFoundError is returned when a resource is not found in the
// processing the query.
type ResourceNotFoundError struct {
	Err          error
	Server       *string
	DatabaseLink *string
	Database     *string
	Schema       *string
	Table        *string
	Column       *string
	Function     *string
}

func (e *ResourceNotFoundError) Error() string {
	var resourceParts []string
	if e.Server != nil {
		resourceParts = append(resourceParts, fmt.Sprintf("server: %s", *e.Server))
	}
	if e.DatabaseLink != nil {
		resourceParts = append(resourceParts, fmt.Sprintf("database link: %s", *e.DatabaseLink))
	}
	if e.Database != nil {
		resourceParts = append(resourceParts, fmt.Sprintf("database: %s", *e.Database))
	}
	if e.Schema != nil {
		resourceParts = append(resourceParts, fmt.Sprintf("schema: %s", *e.Schema))
	}
	if e.Table != nil {
		resourceParts = append(resourceParts, fmt.Sprintf("table: %s", *e.Table))
	}
	if e.Column != nil {
		resourceParts = append(resourceParts, fmt.Sprintf("column: %s", *e.Column))
	}
	if e.Function != nil {
		resourceParts = append(resourceParts, fmt.Sprintf("function: %s", *e.Function))
	}
	parts := []string{
		fmt.Sprintf("resource not found: %s", strings.Join(resourceParts, ", ")),
	}

	if e.Err != nil {
		parts = append(parts, fmt.Sprintf("err: %s", e.Err.Error()))
	}

	return strings.Join(parts, ", ")
}

func (e *ResourceNotFoundError) Unwrap() error {
	return e.Err
}

// TypeNotSupportedError is returned when a type is not supported in the
// query span, for example, using a function as table source.
type TypeNotSupportedError struct {
	Err   error
	Type  string
	Name  string
	Extra string
}

func (e *TypeNotSupportedError) Error() string {
	parts := []string{
		fmt.Sprintf("type not supported: %s", e.Type),
	}

	if e.Err != nil {
		parts = append(parts, fmt.Sprintf("err: %s", e.Err.Error()))
	}

	if e.Name != "" {
		parts = append(parts, fmt.Sprintf("name: %s", e.Name))
	}

	if e.Extra != "" {
		parts = append(parts, fmt.Sprintf("extra: %s", e.Extra))
	}

	return strings.Join(parts, ", ")
}

func (e *TypeNotSupportedError) Unwrap() error {
	return e.Err
}

type FunctionNotSupportedError struct {
	Err        error
	Function   string
	Definition string
}

func (e *FunctionNotSupportedError) Error() string {
	parts := []string{
		fmt.Sprintf("function is not supported with data masking: %s", e.Function),
	}

	if e.Err != nil {
		parts = append(parts, fmt.Sprintf("err: %s", e.Err.Error()))
	}

	if e.Definition != "" {
		parts = append(parts, fmt.Sprintf("definition: %s", e.Definition))
	}

	return strings.Join(parts, ", ")
}
