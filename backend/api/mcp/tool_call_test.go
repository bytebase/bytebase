package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatCallOutput(t *testing.T) {
	endpoint := &EndpointInfo{
		OperationID: "test.Operation",
		Service:     "TestService",
		Method:      "Test",
	}

	output := CallOutput{
		Status:   200,
		Response: map[string]any{"data": "test"},
	}

	result := formatCallOutput(output, endpoint)
	require.Contains(t, result, "test.Operation")
	require.Contains(t, result, "200 OK")
	require.Contains(t, result, "\"data\"")
}

func TestFormatCallOutputError(t *testing.T) {
	endpoint := &EndpointInfo{
		OperationID: "test.Operation",
		Service:     "TestService",
		Method:      "Test",
	}

	output := CallOutput{
		Status: 404,
		Error:  "not found",
	}

	result := formatCallOutput(output, endpoint)
	require.Contains(t, result, "Error")
	require.Contains(t, result, "404")
	require.Contains(t, result, "not found")
}
