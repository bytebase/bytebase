package main

import (
	"fmt"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func loggingReleaseChecks(resp *v1pb.CheckReleaseResponse) error {
	if resp == nil || len(resp.Results) == 0 {
		fmt.Println("No check results found.")
		return nil
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

		if len(advices) > 0 {
			fmt.Printf("%s with %s has %d advices\n", result.File, result.Target, len(advices))
			for _, advice := range result.Advices {
				if advice.Status == v1pb.Advice_WARNING {
					hasWarning = true
				} else if advice.Status == v1pb.Advice_ERROR {
					hasError = true
				}
				fmt.Printf("* (%s) %d - %s: %s\n", advice.Status.String(), advice.Code, advice.Title, advice.Content)
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
		return fmt.Errorf("SQL review failed with errors")
	}
	return nil
}
