package parser

import (
	"fmt"
	"strings"
)

// ResourceNotFoundError is returned when a resource is not found in the
// processing the query.
type ResourceNotFoundError struct {
	Err      error
	Server   *string
	Database *string
	Schema   *string
	Table    *string
	Column   *string
}

func (e *ResourceNotFoundError) Error() string {
	var resourceParts []string
	if e.Server != nil {
		resourceParts = append(resourceParts, fmt.Sprintf("server: %s", *e.Server))
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
	Err  error
	Type string
	Name string
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

	return strings.Join(parts, ", ")
}

func (e *TypeNotSupportedError) Unwrap() error {
	return e.Err
}
