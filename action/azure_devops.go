package main

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func loggingReleaseChecks(resp *v1pb.CheckReleaseResponse, files []*v1pb.Release_File) error {
	if resp == nil || len(resp.Results) == 0 {
		fmt.Println("No check results found.")
		return nil
	}

	fpMap := make(map[string]*v1pb.Release_File)
	for _, file := range files {
		fpMap[file.Path] = file
	}

	hasError := false
	hasWarning := false
	for _, result := range resp.Results {
		var advices []string
		for _, advice := range result.Advices {
			if advice.Status == v1pb.Advice_WARNING || advice.Status == v1pb.Advice_ERROR {
				advices = append(advices, advice.Content)
			}
		}

		fp := result.File
		file := fpMap[fp]
		if len(advices) > 0 {
			fmt.Printf("%s with %s has %d advices\n", result.File, result.Target, len(advices))
			for _, advice := range result.Advices {
				switch advice.Status {
				case v1pb.Advice_WARNING:
					hasWarning = true
				case v1pb.Advice_ERROR:
					hasError = true
				}
				line, column := int(advice.Line), int(advice.Column)
				if file != nil {
					p := common.ConvertPositionToGitHubAnnotationPosition(&storepb.Position{
						Line:   int32(line),
						Column: int32(column),
					}, string(file.Statement))
					line = p.Line
					column = p.Col
				}

				position := fmt.Sprintf("line %d, col %d", line, column)
				fmt.Printf("* (%s) Code %d - %s (%s): %s\n", advice.Status.String(), advice.Code, advice.Title, position, advice.Content)
			}
		}
	}

	if hasError {
		// Azure Pipelines log command.
		fmt.Println("##vso[task.logissue type=error;]SQL review failed with errors")
	} else if hasWarning {
		fmt.Println("##vso[task.logissue type=warning;]SQL review has warnings")
	}

	if hasError {
		return errors.Errorf("SQL review failed with errors")
	}
	return nil
}
