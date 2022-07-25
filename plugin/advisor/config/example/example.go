package main

import (
	_ "embed"
	"fmt"
	"log"

	"github.com/bytebase/bytebase/plugin/advisor/config"
)

//go:embed sql-review-update.yaml
var sqlReviewUpdateStr string

func main() {
	ruleList, err := config.MergeSQLReviewRules(sqlReviewUpdateStr)
	if err != nil {
		log.Fatalf("cannot merge rules with error: %v", err)
	}

	for _, rule := range ruleList {
		fmt.Printf("%+v\n", rule)
	}
}
