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

// ddlDryRunCode is the advisor code for DDL dry run failures.
const ddlDryRunCode = 257

type finding struct {
	file    string
	line    int
	level   v1pb.Advice_Level
	code    int32
	content string
}

func buildCommentMessage(resp *v1pb.CheckReleaseResponse) string {
	var ddlFindings, reviewFindings []finding
	for _, result := range resp.Results {
		for _, advice := range result.Advices {
			if advice.Status == v1pb.Advice_ADVICE_LEVEL_UNSPECIFIED || advice.Status == v1pb.Advice_SUCCESS {
				continue
			}
			f := finding{
				file:    result.File,
				line:    common.ConvertLineToActionLine(int(advice.GetStartPosition().GetLine())),
				level:   advice.Status,
				code:    advice.Code,
				content: advice.Content,
			}
			if advice.Code == ddlDryRunCode {
				ddlFindings = append(ddlFindings, f)
			} else {
				reviewFindings = append(reviewFindings, f)
			}
		}
	}

	var sb strings.Builder
	_, _ = sb.WriteString(commentHeader + "\n")
	_, _ = sb.WriteString("## SQL Review\n\n")
	_, _ = sb.WriteString(fmt.Sprintf("* Affected Rows: **%d**\n", resp.AffectedRows))
	_, _ = sb.WriteString(fmt.Sprintf("* Risk Level: **%s**\n", formatRiskLevel(resp.RiskLevel)))

	if len(ddlFindings) == 0 && len(reviewFindings) == 0 {
		_, _ = sb.WriteString("\nAll checks passed.\n")
		return sb.String()
	}

	if len(ddlFindings) > 0 {
		_, _ = sb.WriteString("\n### DDL Executability\n\n")
		_, _ = sb.WriteString("| File | Line | Error |\n")
		_, _ = sb.WriteString("|------|------|-------|\n")
		for _, f := range ddlFindings {
			if sb.Len() > maxCommentLength-500 {
				_, _ = sb.WriteString("\n_... truncated due to comment length limit._\n")
				return sb.String()
			}
			_, _ = sb.WriteString(fmt.Sprintf("| %s | %d | %s |\n", f.file, f.line, f.content))
		}
	}

	if len(reviewFindings) > 0 {
		_, _ = sb.WriteString("\n### SQL Review Policy\n\n")
		_, _ = sb.WriteString("| File | Line | Level | Finding |\n")
		_, _ = sb.WriteString("|------|------|-------|--------|\n")
		for _, f := range reviewFindings {
			if sb.Len() > maxCommentLength-500 {
				_, _ = sb.WriteString("\n_... truncated due to comment length limit._\n")
				return sb.String()
			}
			_, _ = sb.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", f.file, f.line, formatAdviceLevel(f.level), f.content))
		}
	}

	return sb.String()
}

func formatAdviceLevel(s v1pb.Advice_Level) string {
	switch s {
	case v1pb.Advice_ERROR:
		return "❌"
	case v1pb.Advice_WARNING:
		return "⚠️"
	default:
		return ""
	}
}

func writeAnnotations(resp *v1pb.CheckReleaseResponse) error {
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
			if advice.Code == ddlDryRunCode {
				_, _ = sb.WriteString("DDL Executability")
			} else {
				_, _ = sb.WriteString(advice.Title)
			}
			_, _ = sb.WriteString(" (")
			_, _ = sb.WriteString(strconv.Itoa(int(advice.Code)))
			_, _ = sb.WriteString(")::")
			_, _ = sb.WriteString(advice.Content)
			_, _ = sb.WriteString(". Targets: ")
			_, _ = sb.WriteString(result.Target)
			// Only add doc link for non-DDL dry run codes.
			if advice.Code != ddlDryRunCode {
				_, _ = sb.WriteString(" https://docs.bytebase.com/sql-review/error-codes#")
				_, _ = sb.WriteString(strconv.Itoa(int(advice.Code)))
			}
			fmt.Println(sb.String())
		}
	}
	return nil
}

func formatRiskLevel(r v1pb.RiskLevel) string {
	switch r {
	case v1pb.RiskLevel_LOW:
		return "🟢 Low"
	case v1pb.RiskLevel_MODERATE:
		return "🟡 Moderate"
	case v1pb.RiskLevel_HIGH:
		return "🔴 High"
	case v1pb.RiskLevel_RISK_LEVEL_UNSPECIFIED:
		return "⚪ None"
	default:
		return "⚪ None"
	}
}
