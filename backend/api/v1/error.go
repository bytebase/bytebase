package v1

import (
	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

var (
	// nonSelectSQLError returns an error indicating a non-SELECT SQL error.
	nonSelectSQLError = func() *connect.Error {
		err := connect.NewError(connect.CodeInvalidArgument, errors.New("Support SELECT sql statement only"))
		detail, detailErr := connect.NewErrorDetail(&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "statement",
					Description: "statement must be read-only SELECT statement",
				},
			},
		})
		if detailErr != nil {
			panic(detailErr)
		}
		err.AddDetail(detail)
		return err
	}()
	// nonReadOnlyCommandError returns an error indicating a non-read-only command error.
	nonReadOnlyCommandError = func() *connect.Error {
		err := connect.NewError(connect.CodeInvalidArgument, errors.New("Support read-only command statements only"))
		detail, detailErr := connect.NewErrorDetail(&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "statement",
					Description: "statement must be read-only command statement",
				},
			},
		})
		if detailErr != nil {
			panic(detailErr)
		}
		err.AddDetail(detail)
		return err
	}()
)
