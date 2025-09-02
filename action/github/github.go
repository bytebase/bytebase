package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const maxCommentLength = 65536
const commentHeader = `<!--BYTEBASE_MARKER-DO_NOT_EDIT-->`
const githubActionUserID = 41898282

type githubEnv struct {
	APIUrl string `env:"GITHUB_API_URL,required,notEmpty"`
	Repo   string `env:"GITHUB_REPOSITORY,required,notEmpty"`
	Token  string `env:"GITHUB_TOKEN,required,notEmpty"`

	EventName string `env:"GITHUB_EVENT_NAME,required,notEmpty"`
	EventPath string `env:"GITHUB_EVENT_PATH,required,notEmpty"`
}

func CreateCommentAndAnnotation(resp *v1pb.CheckReleaseResponse) error {
	// Write annotations to the pull request.
	if err := writeAnnotations(resp); err != nil {
		return err
	}

	ghe, err := env.ParseAs[githubEnv]()
	if err != nil {
		return errors.Wrap(err, "failed to parse GitHub environment variables")
	}
	// Upsert a comment on the pull request with the check results.
	if err := upsertComment(resp, &ghe); err != nil {
		fmt.Printf("failed to upsert comment on the pull request: %v\n", err)
		return nil
	}
	return nil
}

func getPRNumberFromEventFile(eventPath string) (string, error) {
	eventFile, err := os.ReadFile(eventPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read event file %s", eventPath)
	}
	var event struct {
		Number int64 `json:"number"`
	}
	if err := json.Unmarshal(eventFile, &event); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal event file %s", eventPath)
	}
	if event.Number == 0 {
		return "", errors.New("no pull request number found in the event file")
	}
	return fmt.Sprintf("%d", event.Number), nil
}

func upsertComment(resp *v1pb.CheckReleaseResponse, ghe *githubEnv) error {
	if ghe.EventName != "pull_request" {
		fmt.Println("::warning not a pull request event, will not create a comment.")
		return nil
	}
	pr, err := getPRNumberFromEventFile(ghe.EventPath)
	if err != nil {
		fmt.Printf("::warning failed to get pull request number from event file, will not create a comment. error: %v", err.Error())
		return nil
	}
	if pr == "" {
		fmt.Println("::warning no pull request number found in the environment variables, will not create a comment.")
		return nil
	}
	// upsert the comment
	c := newClient(ghe.APIUrl, ghe.Token)
	comments, err := c.listComments(ghe.Repo, pr)
	if err != nil {
		return errors.Wrapf(err, "failed to list comments")
	}
	for _, comment := range comments {
		if comment.User.ID == githubActionUserID && strings.HasPrefix(comment.Body, commentHeader) {
			// update the comment
			if err := c.updateComment(ghe.Repo, comment.ID, buildCommentMessage(resp)); err != nil {
				return errors.Wrapf(err, "failed to update comment")
			}
			return nil
		}
	}

	// create a new comment
	if err := c.createComment(ghe.Repo, pr, buildCommentMessage(resp)); err != nil {
		return errors.Wrapf(err, "failed to create comment")
	}
	return nil
}

func buildCommentMessage(resp *v1pb.CheckReleaseResponse) string {
	var errorCount, warningCount int
	for _, result := range resp.Results {
		for _, advice := range result.Advices {
			switch advice.Status {
			case v1pb.Advice_WARNING:
				warningCount++
			case v1pb.Advice_ERROR:
				errorCount++
			case v1pb.Advice_STATUS_UNSPECIFIED, v1pb.Advice_SUCCESS:
				// No action needed
			default:
				// Ignore unknown advice statuses
			}
		}
	}

	var sb strings.Builder
	_, _ = sb.WriteString(commentHeader + "\n")
	_, _ = sb.WriteString("## SQL Review Summary\n\n")
	_, _ = sb.WriteString(fmt.Sprintf("* Total Affected Rows: **%d**\n", resp.AffectedRows))
	_, _ = sb.WriteString(fmt.Sprintf("* Overall Risk Level: **%s**\n", formatRiskLevel(resp.RiskLevel)))
	_, _ = sb.WriteString(fmt.Sprintf("* Advices Statistics: **%d Error(s), %d Warning(s)**\n", errorCount, warningCount))
	_, _ = sb.WriteString("### Detailed Results\n")
	_, _ = sb.WriteString(`
<table>
  <thead>
    <tr>
      <th>File</th>
      <th>Target</th>
      <th>Affected Rows</th>
      <th>Risk Level</th>
      <th>Advices</th>
    </tr>
  </thead>
  <tbody>`)
	for _, result := range resp.Results {
		if sb.Len() > maxCommentLength-1000 {
			break
		}
		var errorCount, warningCount int
		for _, advice := range result.Advices {
			switch advice.Status {
			case v1pb.Advice_WARNING:
				warningCount++
			case v1pb.Advice_ERROR:
				errorCount++
			case v1pb.Advice_STATUS_UNSPECIFIED, v1pb.Advice_SUCCESS:
				// No action needed
			default:
				// Ignore unknown advice statuses
			}
		}
		counts := []string{}
		if errorCount > 0 {
			counts = append(counts, fmt.Sprintf("%d Error(s)", errorCount))
		}
		if warningCount > 0 {
			counts = append(counts, fmt.Sprintf("%d Warning(s)", warningCount))
		}
		adviceCell := "-"
		if len(counts) > 0 {
			adviceCell = strings.Join(counts, ", ")
		}

		_, _ = sb.WriteString(fmt.Sprintf(`<tr>
<td>%s</td>
<td>%s</td>
<td>%d</td>
<td>%s</td>
<td>%s</td>
</tr>`, result.File, result.Target, result.AffectedRows, formatRiskLevel(result.RiskLevel), adviceCell))
	}
	_, _ = sb.WriteString("</tbody></table>")
	return sb.String()
}

func writeAnnotations(resp *v1pb.CheckReleaseResponse) error {
	// annotation template
	// `::${advice.status} file=${file},line=${advice.line},col=${advice.column},title=${advice.title} (${advice.code})::${advice.content}. Targets: ${targets.join(', ')} https://docs.bytebase.com/sql-review/error-codes#${advice.code}`
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
			_, _ = sb.WriteString(strconv.Itoa(common.ConvertLineToActionLine(int(advice.GetStartPosition().GetLine()))))
			_, _ = sb.WriteString(",title=")
			_, _ = sb.WriteString(advice.Title)
			_, _ = sb.WriteString(" (")
			_, _ = sb.WriteString(strconv.Itoa(int(advice.Code)))
			_, _ = sb.WriteString(")::")
			_, _ = sb.WriteString(advice.Content)
			_, _ = sb.WriteString(". Targets: ")
			_, _ = sb.WriteString(result.Target)
			_, _ = sb.WriteString(" ")
			_, _ = sb.WriteString(" https://docs.bytebase.com/sql-review/error-codes#")
			_, _ = sb.WriteString(strconv.Itoa(int(advice.Code)))
			fmt.Println(sb.String())
		}
	}
	return nil
}

func formatRiskLevel(r v1pb.CheckReleaseResponse_RiskLevel) string {
	switch r {
	case v1pb.CheckReleaseResponse_LOW:
		return "🟢 Low"
	case v1pb.CheckReleaseResponse_MODERATE:
		return "🟡 Moderate"
	case v1pb.CheckReleaseResponse_HIGH:
		return "🔴 High"
	case v1pb.CheckReleaseResponse_RISK_LEVEL_UNSPECIFIED:
		return "⚪ None"
	default:
		return "⚪ None"
	}
}
