// Package aiaccess defines the capability vocabulary that AI surfaces resolve
// every API method against.
//
// One evaluator governs AI access: required(request) ⊆ resolve(ceiling). This
// package owns both halves' vocabulary — what a method requires, and what the
// predefined ceilings contain — and nothing else. It performs no enforcement;
// the gate that consumes it lands in a later change.
package aiaccess

import (
	"fmt"
	"slices"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	// Registers every bytebase.v1 file descriptor in the global proto registry,
	// which is what V1Procedures walks.
	_ "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// v1Package is the proto package whose RPCs this policy governs.
const v1Package = "bytebase.v1"

// V1Procedures returns every bytebase.v1 RPC as a connect procedure name
// ("/bytebase.v1.SQLService/Query"), sorted.
//
// The universe comes from the proto registry rather than a hand-kept list so a
// newly added RPC is visible here the moment it is generated, which is what
// lets the exactly-once lint fail closed on it.
func V1Procedures() []string {
	var procedures []string
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != v1Package {
			return true
		}
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			methods := service.Methods()
			for j := 0; j < methods.Len(); j++ {
				procedures = append(procedures, fmt.Sprintf("/%s/%s", service.FullName(), methods.Get(j).Name()))
			}
		}
		return true
	})
	slices.Sort(procedures)
	return procedures
}
