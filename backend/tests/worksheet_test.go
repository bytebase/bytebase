package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestSearchWorksheetsPagination(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	var worksheetNames []string
	for _, title := range []string{"pagination worksheet 1", "pagination worksheet 2"} {
		resp, err := ctl.worksheetServiceClient.CreateWorksheet(ctx, connect.NewRequest(&v1pb.CreateWorksheetRequest{
			Parent: ctl.project.Name,
			Worksheet: &v1pb.Worksheet{
				Title:      title,
				Content:    []byte("SELECT 1;"),
				Visibility: v1pb.Worksheet_PRIVATE,
			},
		}))
		a.NoError(err)
		worksheetNames = append(worksheetNames, resp.Msg.Name)
	}

	firstPage, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent:   ctl.project.Name,
		Filter:   `creator == "users/demo@example.com"`,
		PageSize: 1,
	}))
	a.NoError(err)
	a.Len(firstPage.Msg.Worksheets, 1)
	a.NotEmpty(firstPage.Msg.NextPageToken)

	secondPage, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent:    ctl.project.Name,
		Filter:    `creator == "users/demo@example.com"`,
		PageSize:  1,
		PageToken: firstPage.Msg.NextPageToken,
	}))
	a.NoError(err)
	a.Len(secondPage.Msg.Worksheets, 1)
	a.Empty(secondPage.Msg.NextPageToken)

	gotNames := []string{firstPage.Msg.Worksheets[0].Name, secondPage.Msg.Worksheets[0].Name}
	a.ElementsMatch(worksheetNames, gotNames)
}
