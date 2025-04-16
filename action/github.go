package main

import (
	"fmt"
	"strings"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func writeAnnotations(resp *v1pb.CheckReleaseResponse) error {
	// annotation template
	// `::${advice.status} file=${file},line=${advice.line},col=${advice.column},title=${advice.title} (${advice.code})::${advice.content}. Targets: ${targets.join(', ')} https://www.bytebase.com/docs/reference/error-code/advisor#${advice.code}`
	for _, result := range resp.Results {
		for _, advice := range result.Advices {
			var sb strings.Builder
			_, _ = sb.WriteString("::")
			switch advice.Status {
			case v1pb.Advice_WARNING:
				_, _ = sb.WriteString("warning ")
			case v1pb.Advice_ERROR:
				_, _ = sb.WriteString("error ")
			default:
				continue
			}

			_, _ = sb.WriteString(" file=")
			_, _ = sb.WriteString(result.File)
			_, _ = sb.WriteString(",line=")
			_, _ = sb.WriteString(string(advice.Line))
			_, _ = sb.WriteString(",col=")
			_, _ = sb.WriteString(string(advice.Column))
			_, _ = sb.WriteString(",title=")
			_, _ = sb.WriteString(advice.Title)
			_, _ = sb.WriteString(" (")
			_, _ = sb.WriteString(string(advice.Code))
			_, _ = sb.WriteString(")::")
			_, _ = sb.WriteString(advice.Content)
			_, _ = sb.WriteString(". Targets: ")
			_, _ = sb.WriteString(result.Target)
			_, _ = sb.WriteString(" ")
			_, _ = sb.WriteString(" https://www.bytebase.com/docs/reference/error-code/advisor#")
			_, _ = sb.WriteString(string(advice.Code))
			fmt.Println(sb.String())
		}
	}
	return nil
}
