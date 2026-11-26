package drivers

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	odriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func stringValue(s string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: s}}
}

// printedValue is a cell the editor shows as Go prints the contract's value.
func printedValue(value odriver.Value) *v1pb.RowValue {
	return stringValue(fmt.Sprintf("%v", value))
}

// jsonValue is a structured cell as JSON text.
func jsonValue(value odriver.Value) *v1pb.RowValue {
	encoded, err := json.Marshal(value.Structured)
	if err != nil {
		return stringValue(fmt.Sprintf("%v", value.Structured))
	}
	return stringValue(string(encoded))
}

// joinHostPort joins host and port, accepting a host already in brackets.
func joinHostPort(host, port string) string {
	return net.JoinHostPort(strings.TrimPrefix(strings.TrimSuffix(host, "]"), "["), port)
}
