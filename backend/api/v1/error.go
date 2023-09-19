package v1

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// nonSelectSQLError returns an error indicating a non-SELECT SQL error.
	nonSelectSQLError, _ = status.New(codes.InvalidArgument, "Support SELECT sql statement only").WithDetails(
		&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "statement",
					Description: "statement must be read-only SELECT statement",
				},
			},
		},
	)
)
